package engine

import (
	"errors"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"interdoor.net/interdoor/internal/engine/term"
)

// Server accepts SSH connections and runs the registered game for each session.
// SSH-level auth is open (BBS/door model); players authenticate in-game with a
// name and password.
type Server struct {
	addr            string
	cfg             *ssh.ServerConfig
	store           *Store
	game            Game
	nodeID          string
	maxSessions     int
	idleTimeout     time.Duration
	ln              net.Listener
	closing         atomic.Bool
	sessions        atomic.Int32
	crossNodeAttack CrossNodePvPFn
	travelFn        TravelFn
}

// SetCrossNodeAttack injects the federation PvP submission function. Call before
// ListenAndServe when a hub is configured.
func (s *Server) SetCrossNodeAttack(fn CrossNodePvPFn) { s.crossNodeAttack = fn }

// SetTravelFn injects the cross-node travel function. Call before ListenAndServe.
func (s *Server) SetTravelFn(fn TravelFn) { s.travelFn = fn }

// Options configures a node server.
type Options struct {
	Addr        string
	HostKey     ssh.Signer
	NodeID      string
	MaxSessions int           // 0 = unlimited
	IdleTimeout time.Duration // 0 = no idle timeout
}

// NewServer builds a node server from options, serving game, persisting to store.
func NewServer(store *Store, game Game, opt Options) *Server {
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(opt.HostKey)
	return &Server{
		addr: opt.Addr, cfg: cfg, store: store, game: game, nodeID: opt.NodeID,
		maxSessions: opt.MaxSessions, idleTimeout: opt.IdleTimeout,
	}
}

// ListenAndServe blocks, accepting connections until Shutdown is called or the
// listener errors. A clean shutdown returns nil.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.ln = ln
	for {
		nConn, err := ln.Accept()
		if err != nil {
			if s.closing.Load() {
				return nil // expected: Shutdown closed the listener
			}
			return err
		}
		go s.handleConn(nConn)
	}
}

// Shutdown stops accepting new connections. In-flight sessions are left to finish
// (process exit tears them down); graceful per-session drain is future work.
func (s *Server) Shutdown() error {
	s.closing.Store(true)
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) handleConn(nConn net.Conn) {
	defer nConn.Close()
	conn, chans, reqs, err := ssh.NewServerConn(nConn, s.cfg)
	if err != nil {
		return // handshake failed; nothing to do
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		if s.maxSessions > 0 && int(s.sessions.Load()) >= s.maxSessions {
			_ = newCh.Reject(ssh.ResourceShortage, "The Low is full tonight. Try again shortly.")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			continue
		}
		s.sessions.Add(1)
		go s.serveSession(ch, chReqs)
	}
}

// serveSession waits for the client's shell request, then runs the app.
func (s *Server) serveSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	defer s.sessions.Add(-1)
	// A panic in game logic must not take down the node — contain it to the session.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("session panic recovered: %v", r)
		}
	}()

	ready := make(chan struct{}, 1)
	go func() {
		for req := range reqs {
			switch req.Type {
			case "pty-req", "window-change", "env":
				// Accepted but unused — the playfield is a fixed 80x24.
				_ = req.Reply(true, nil)
			case "shell":
				_ = req.Reply(true, nil)
				select {
				case ready <- struct{}{}:
				default:
				}
			default:
				_ = req.Reply(false, nil)
			}
		}
	}()

	<-ready
	t := term.New(ch, ch)

	if s.idleTimeout > 0 {
		done := make(chan struct{})
		defer close(done)
		go s.idleWatch(ch, t, done)
	}

	if err := s.runApp(t); err != nil && !errors.Is(err, errQuit) {
		log.Printf("session ended: %v", err)
	}
}

// idleWatch closes the channel if the client sends no input for idleTimeout,
// reclaiming the session slot from dead connections.
func (s *Server) idleWatch(ch ssh.Channel, t *term.Terminal, done <-chan struct{}) {
	interval := s.idleTimeout / 2
	if interval < time.Second {
		interval = time.Second
	}
	tk := time.NewTicker(interval)
	defer tk.Stop()
	for {
		select {
		case <-done:
			return
		case <-tk.C:
			if time.Since(t.LastActivity()) > s.idleTimeout {
				_ = ch.Close() // unblocks the session's pending read, ending it
				return
			}
		}
	}
}

var errQuit = errors.New("quit")

// runApp presents the front-door login/create flow, then runs the game.
func (s *Server) runApp(t *term.Terminal) error {
	p, err := s.login(t)
	if err != nil {
		return err
	}
	_ = s.store.TouchLastSeen(p.GlobalID)
	ctx := &Context{
		Player: p, Term: t, Store: s.store, NodeID: s.nodeID,
		CrossNodeAttack: s.crossNodeAttack,
		Travel:          s.travelFn,
	}
	return s.game.Run(ctx)
}

func (s *Server) login(t *term.Terminal) (*Player, error) {
	for {
		f := term.NewFrame()
		f.Blank()
		for _, line := range strings.Split(s.game.Banner(), "\n") {
			f.Line(line)
		}
		f.Blank()
		f.Line("    [L] Log in")
		f.Line("    [C] Create a new wanderer")
		f.Line("    [Q] Disconnect")
		f.Status("    Choose:  (an InterDOOR node)")
		f.Render(t)

		key, err := t.ReadKey()
		if err != nil {
			return nil, err
		}
		switch key {
		case 'l', 'L':
			if p := s.doLogin(t); p != nil {
				return p, nil
			}
		case 'c', 'C':
			if p := s.doCreate(t); p != nil {
				return p, nil
			}
		case 'q', 'Q':
			return nil, errQuit
		}
	}
}

func (s *Server) doLogin(t *term.Terminal) *Player {
	t.Clear()
	t.Write("\r\n  Name: ")
	name, err := t.ReadLine(true)
	if err != nil {
		return nil
	}
	t.Write("  Password: ")
	pw, err := t.ReadLine(false)
	if err != nil {
		return nil
	}
	p, err := s.store.Authenticate(name, pw)
	if err != nil {
		t.Write("\r\n  " + term.Bright(term.Red) + "No such name and password." + term.Reset() + " (press any key)")
		_, _ = t.ReadKey()
		return nil
	}
	if p.Status == "traveling" {
		t.Write("\r\n  " + term.Bright(term.Yellow) + "Your wanderer walks the road between nodes." + term.Reset() + "\r\n  They are not here right now. (press any key)")
		_, _ = t.ReadKey()
		return nil
	}
	return p
}

func (s *Server) doCreate(t *term.Terminal) *Player {
	t.Clear()
	t.Write("\r\n  Choose a name (3-16 chars): ")
	name, err := t.ReadLine(true)
	if err != nil {
		return nil
	}
	t.Write("  Choose a password (4+ chars): ")
	pw, err := t.ReadLine(false)
	if err != nil {
		return nil
	}
	p, err := s.store.CreateAccount(name, pw)
	if err != nil {
		t.Write("\r\n  " + term.Bright(term.Red) + err.Error() + term.Reset() + " (press any key)")
		_, _ = t.ReadKey()
		return nil
	}
	if err := s.game.NewCharacter(s.store.DB(), p); err != nil {
		log.Printf("NewCharacter failed for %s: %v", p.GlobalID, err)
	}
	return p
}
