{Door Game..by Benson Wong}
{Version .01b}
{Notes:
  When the game begins the player starts with the following:
    $20,000                  (DEFAULT, SYSOP DEFINED)
    1 building points        (DEFAULT, SYSOP DEFINED)
    1 research points        (DEFAULT, SYSOP DEFINED)
    27,426 population        (DEFAULT, SYSOP DEFINED)

    100 Normal type soldiers (DEFAULT, SYSOP DEFINED)
    10 Agricultrial Region   (DEFAULT, SYSOP DEFINED)
    3 Industrial Regions     (DEFAULT, SYSOP DEFINED)
    10,000 Food               (DEFAULT, SYSOP DEFINED)
    ALL OTHER THINGS ARE SET TO ZERO
}
USES crt,ddplus,Dos;

Const
  Player_Data_Filename='PLAYERS.DAT'; {Players save file}
  Problem_Log_FileName='PROBLEMS.LOG';
  Message_Outfile_Name='Messages.dat';
  Registered=False;
  NewsFileName='GAMENEWS.ANS';

Type
  Global_E = Object              {These are the variables that control how}
                                 {many turns the player has and what the new}
                                 {players start out with}
    Set_Turns:integer;           {Sets how many month or turns the player has}
                                 {every day}
  {Misc}
    Set_Money:longint;           {Sysop defined money variable}
    Set_MoneyBank:longint;       {Sysop defined money in bank variable}
    Set_Population:longint;      {Sysop defined Population}
    Set_Food:longint;            {Sysop defined food}
    Set_Energy:longint;          {Sysop defined energy}
    Set_BuildingP:longint;
    Set_ResearchP:longint;
  {Military}
    Set_HumanS:longint;          {Sysop defined Human soldiers}
  {Regions}
    Set_Agricultural:longint;    {Sysop defined Agricultural Regions}
    Set_Industrial:longint;      {Sysop defined Industrial Regions}
    Set_Desert:longint;          {Sysop defined Desert Regions}
    Set_Urban:longint;           {Sysop defined Urban Regions}
    Set_Volcanic:longint;        {Sysop defined Volcanic Regions}
    Set_River:longint;           {Sysop defined River Regions}
    Set_OceanSea:longint;        {Sysop defined Ocean And Sea Regions}
    Set_WasteLand:longint;       {Sysop defined Wasteland Regions}
  {Technology}
   {Energy Production Technology }
    Set_HydroT:boolean;          {Sysop defined: Hydro Power Technology }
    Set_WindT:boolean;           {Sysop Defined: Wind Power technology}
    Set_SolarT:boolean;          {Sysop Defined: Solar Power Technology}
    Set_FossilT:boolean;         {Sysop Defined: Fossil Fuel, Default: TRUE}

    Set_FissionT:boolean;        {Sysop Defined: Fission Power Technology}
       Set_FusionT:boolean;      {Sysop Defined: Fusion Power Technology}
          Set_CFusionT:boolean;  {Sysop Defined: Cold Fusion Power Technology}
   {---- War Technology ----}
 {Soldier Technology}
    Set_HumanNT:boolean;         {Sysop Defined: Normal Human Technology,Default:True}
    Set_GAHumanT:boolean;        {Sysop Defined: GAHuman Technology}
    Set_SuperHumanT:boolean;     {Sysop Defined: SuperHuman Technology}
    Set_HyperHumanT:boolean;     {Sysop Defined: HyperHuman Technology}
    Set_MegaHumanT:boolean;      {Sysop Defined: MegaHuman Technology}
 {BioBot Upgrade Technology}
    Set_BasicBT:boolean;         {Sysop Defined: Basic BioBot Technology}
    Set_StandardBT:boolean;      {Sysop Defined: Standard BioBot Technology}
    Set_AdvancedBT:boolean;      {Sysop Defined: Advanced BioBot Technology}
    Set_Advanced2BT:boolean;     {Sysop Defined: Advanced BioBot L2 Technology}
    Set_BattleBT:boolean;        {Sysop Defined: Battle BioBot Technology}
    Set_HeavyAttackBT:boolean;   {Sysop Defined: Heavy Attack BioBot Technology}
    Set_OmegaBT:boolean;         {Sysop Defined: OMega BioBot Technology}
    Set_OmegaL2BT:boolean;       {Sysop Defined: Omega BioBot L2 Technology}
 {Drone Technology}
    Set_Recon1T:boolean;         {Sysop Defined: Recon Level 1 Droid}
    Set_Recon2T:boolean;         {Sysop Defined: Recon Level 2 Droid}
    Set_Recon3T:boolean;         {Sysop Defined: Recon Level 3 Droid}
 {War Vehical Technology}
    Set_TankT:boolean;           {Sysop Defined: Normal Tank Technology}
    Set_HoverCT:boolean;         {Sysop Defined: HoverCraft Technology}
    Set_JetT:boolean;            {Sysop Defined: Jet Technology}
 {War Ballistic Weapon Technology}
    Set_NukeBT:boolean;          {Sysop Defined: Nuclear Bomb Technology}
    Set_RadiationBT:boolean;     {Sysop Defined: Radation Bomb Technology}
 {Viral Weapons}
    Set_HypnoT:Boolean;          {Sysop Defined: Hypnobomb Technology}
    Set_PopulationBT:boolean;    {Sysop Defined: Population Bomb Technology}
    Set_FoodBT:boolean;          {Sysop Defined: Food Bomb Technology}
    Set_AntimatterT:boolean;     {Sysop Defined: AntiMatter Bomb Technology}
 {---- Global Defence Technology ----}
  {ODS=Orbital Defence Satillite, ODSNS = Oribital Defence Satillte Network Software}
    Set_ODST:boolean;                {Sysop Defined: ODST Technology}
    Set_ODSNST:boolean;              {Sysop Defined: ODSNST Technology}
    Set_GroundTT:boolean;            {Sysop Defined: Ground Turret Technology}
    Set_AntiT:boolean;               {Sysop Defined: Anti Aircraft Technology}
    Set_GSGT:boolean;                {Sysop Defined: Global Shield Generator Technology}
    Set_DoomsDayT:boolean;           {Sysop Defined: DoomsDay Bomb Technology}

  {-=-=-=-=-=- Sysop Defined Players Army Startup Status -=-=-=-=-=-=-=-}
       {Soldier Storage Variables}
       {Human Type Soldiers}
         Set_HumanN:longint;        {Number of normal human soldiers}
         Set_GAHuman:longint;       {Number of GAHuman soldiers}
         Set_SuperHuman:longint;    {Number of Super Human soldiers}
         Set_HyperHuman:longint;    {Number of HyperHuman soldiers}
         Set_MegaHuman:longint;     {Number of MegaHuman soldiers}

         Set_BasicB_U:longint;      {Number of Basic BioBot Upgraded Soldiers}
         Set_StandardB_U:longint;   {Number of Standard BioBot Upgraded Soldiers}
         Set_AdvancedB_U:longint;   {Number of Advanced BioBot Upgraded Soldiers}
         Set_AdvancedB2_U:longint;  {Number of Advanced-2 BioBot Upgraded Soldiers}
         Set_BattleB_U:longint;     {Number of Battle BioBot Upgraded Soldiers}
         Set_HeavyAttackB_U:longint;{Number of Heavy Attack BioBot upgraded Soldiers}
         Set_OmegaB_U:longint;      {Number of Omega BioBot upgraded soldiers}
         Set_OmegaBL2_U:Longint;    {Number of Omega L2 BioBot upgraded soldiers}

       {Ballistic Attack Weapons}
         Set_AntiMatter_Q:longint;     {Number of Antimatter Missles The Player Has}
         Set_FoodB_Q:longint;          {Number of Food Bombs The Player Has}
         Set_NukeB_Q:longint;          {Number of nuclear missles the player has}
         Set_RadiationB_Q:longint;     {Number of Radiation Bombs the player has}
         Set_PopulationB_Q:longint;    {Number Of Population Bombs The Player Has}
         Set_HypnoB_Q:longint;         {Number Of HypnoBombs the player has}

       {Player's Recon Drone}
         Set_Recon1_Q:longint;     {Player starts out with this technology}
         Set_Recon2_Q:longint;     {Did the player research Recon-2 Drone Tech.}
         Set_Recon3_Q:longint;     {Did the player research Recon-3 Drone Tech.}

       {Player's War Vehicals}
         Set_Tank_Q:longint;        {Did the player research tank tech.}
         Set_HoverC_T:longint;      {Did the player research Hover Craft Tech.}
         Set_Jet_T:longint;         {Did the player research Jet Tech.}

       {Misc / Advanced Buildings...and other variables for them}
         Set_Miners_Guild:boolean;  {Does the player have the Miners guild?}
           Set_NumMiners:longint;   {Number of miners that are unassigned to a mine}

         Set_Fishing_Guild:boolean; {Does the player has the fishing guild?}
           Set_NumFishers:longint;  {The amount of fishermen the player has}
           Set_FishStock:longint;   {The amount of fish remaining}

         Set_Construction_B:longint;{Number of construction sites the player has}
         Set_Labs_B:longint;        {Number of labs the player has}

         Set_Intell_B:boolean;      {Does the player have the Intelligence building}
           Set_Num_Spy:longint;     {Number of spies the player has}
           Set_Terrorists:longint;  {Number of terrorists the player has}

         Set_Lottery_B:boolean;     {Does the player have the lottery set up}
      {---Global Defence Systems---}
        Set_ODSnum:longint;         {How many defence satillites the player has}
        Set_ODSNS:boolean;          {Is the player using the ODSNS?}
        Set_GSGnum:longint;         {How many Global Shield Generators the player has}
        Set_GroundTNum:longint;     {How many Ground Turrets The Player Has}
        Set_AntiANum:longint;       {How many Anti-Aircraft Missle launchers Player has}
        Set_DoomsDay_B:boolean;     {Does the player have DDay weapon Built}
     {---- MINES -------}
        Set_GOLD:longint;           {Sysop Defined Gold Mines}
        Set_silver:longint;           {player's silver mines}
        Set_Iron:longint;             {player's iron mines}
        Set_nickel:longint;           {player's nickel mines}
        Set_copper:longint;           {player's copper mines}
  end;

  Message_Type = record                 {Type used for messages}
    Empire:string[20];                   {Record # of Reciever}
    Message:array[1..15] of string[75]; {The Message}
  end;

   Region_Type=Record        {A Region Record Template to keep track of regions}
      Quantity:longint;      {Quantity of this type the player has}
      Ready_reg:longint;     {Cost to Ready Region}
      Num_Used:longint;      {Number of regions already used}
   end;

 {-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=}

   Mines=record    {Used for the different types of mines}
     Num_mines:longint;      {Number of this type of mine the player has}
     Miners:longint;         {Number of miners at the current mine type}
     MinLeft:longint;        {Amount of mineral left in mines}
   end;

 {-=-=-=-=-=-=-=The Users Empire..this is gonna be big!-=-=-=-=-=-=-=-=-=}

   One_Player=record       {One record for each player}
       Last_On:integer;             {The last date the player was on}
       Turns_Left:byte;             {The Number Of Turns The Player Has Left}
       Players_1stName:string[30];  {The players first name}
       Players_LstName:string[30];  {The players last name}
       Worlds_Name:string[20];      {The Name Of The Players Planet}
       Empire_name:string[20];      {The Name of the players empire}
     {Comparison Variables For Player}
       Worlds_NameC:string[20];     {Search variable For Player's Worlds Name}
       Empires_NameC:string[20];    {Search Variable for Players Empires Name}
       {Main Player Variables}
       {Researching and Building Points}
         Research_P:longint;     {Amount of research points the player has}
         Building_P:longint;     {Amount of building points the player has}

       {POPULATION & FOOD}
         Population:longint;     {Population of Empire}
         Food:longint;           {Amount of food the player has on hand}
         Food_Storage:longint;   {Amount of food the player has in storage}

       {MONEY}
         Money:longint;          {Money value the player has for immediate use}
         Money_bank:longint;     {Money the player has in the bank}

       {ENERGY}
         Energy:longint;         {Energy value the player has for immediate use}
{         Energy_Cores:shortint;  {Number of energy cores the player has Max:10}
         Energy_bank:array[1..10] of longint;
         {^^ amount of Energy the player has in each core}

       {Number of Different Types Of Power Generation Buildings}
         Hydro_B:longint;        {Number of hydro plants player has}
         Wind_B:longint;         {Number of Wind Turbines the player has}
         Solar_B:longint;        {Number of Solar panels the player has}
         Fossil_B:longint;       {Number of Fossil Fuel Plants}
         GeoTher_B:longint;      {Number of GeoThermic Taps the player has}

       {Nuclear Energy}
         Fission_B:longint;      {Number of fission plants the player has}
         Fusion_B:longint;       {Number of fusion plants the player has}
         CFusion_B:longint;      {Number of cold fusion plants the player has}

       {Regions -Stores information on players regions}
         Agricult:Region_Type;   {Agricultural Region}
         Desert:Region_Type;     {Desert Region}
         Industrial:Region_Type; {Industrial Region}
         Urban:Region_Type;      {Urban Region}
         Volcanic:Region_Type;   {Volcanic Region}
         River:Region_Type;      {River Region}
         OceanSea:Region_type;   {Ocean/Sea region}
         Wastelands:longint;     {Completely useless region}

       {Mines}
         Gold:mines;             {player's gold mines}
         silver:mines;           {player's silver mines}
         Iron:mines;             {player's iron mines}
         nickel:mines;           {player's nickel mines}
         copper:mines;           {player's copper mines}

       {-=-=-=-=-=-= Players Researched Technology -=-=-=-=-=-=-}
       {Energy Production Technology }
         Hydro_T:boolean;       {Did the player research hydro Tech.}
         Wind_T:boolean;        {Did the player research Wind Tech.}
         Solar_T:boolean;       {Did the player research Solar Tech.}
         Fossil_T:boolean;      {Player gets this when starting out}

         Fission_T:boolean;     {Did the player research Fission Tech.}
           Fusion_T:boolean;    {Did the player research Fusion Tech.}
             CFusion_T:boolean; {Did the player research Cold Fusion Tech.}

       {---- War Technology ----}
       {Soldier Technology}
         HumanN_T:boolean;      {Player gets this when starting out}
         GAHuman_T:boolean;     {Did the player research GAHuman Tech.}
         SuperHuman_T:boolean;  {Did the player research SuperHuman Tech.}
         HyperHuman_T:boolean;  {Did the player research HyperHuman Tech.}
         MegaHuman_T:boolean;   {Did the player research MegaHuman Tech.}
       {BioBot Upgrade Technology}
         BasicB_T:boolean;      {Did the player research basic BioBot Tech.}
         StandardB_T:boolean;   {Did the player research standard BioBot Tech.}
         AdvancedB_T:boolean;   {Did the player research Advanced BioBot Tech}
         Advanced2B_T:boolean;  {Did the player research Advanced 2 BioBot Tech.}
         BattleB_T:boolean;     {Did the player research Battle BioBot Tech.}
         HeavyAttackB_T:boolean;{Did the player research Heavy Attack BioBot Tech.}
         OmegaB_T:boolean;      {Did the player research Omega BioBot Tech.}
         OmegaL2B_T:boolean;    {Did the player research Omega L2 BioBot Tech.}
       {Drone Technology}
         Recon1_T:boolean;     {Player starts out with this technology}
         Recon2_T:boolean;     {Did the player research Recon-2 Drone Tech.}
         Recon3_T:boolean;     {Did the player research Recon-3 Drone Tech.}
       {War Vehical Technology}
         Tank_T:boolean;        {Did the player research tank tech.}
         HoverC_T:boolean;      {Did the player research Hover Craft Tech.}
         Jet_T:boolean;         {Did the player research Jet Tech.}

       {War Ballistic Weapon Technology}
         NukeB_T:boolean;          {Did the player research Nuclear bomb tech.}
         RadiationB_T:boolean;     {Did the player research Radiation Bomb tech.}
         Antimatter_T:boolean;     {Did the player research Antimatter bomb tech.}

       {Viral Weapons}
         PopulationB_T:boolean;    {Did the player research Population Bomb tech.}
         HypnoB_T:boolean;         {Did the player research HypnoBomb Technology}
         FoodB_T:boolean;          {Did the player research Food Bomb tech.}

    {---- Global Defence Technology ----}
         {ODS=Orbital Defence Satillite, ODSNS = Oribital Defence Satillte Network Software}
         ODS_T:boolean;            {Did the player research ODS tech.}
         ODSNS_T:boolean;          {Did the player research ODSN tech.}

         GroundT_T:boolean;        {Does the player have Ground Turret Tech.}
         Anti_T:boolean;           {Does the player have anti-aircraft tech.}

         GSG_T:boolean;            {Does the player have Global Shield Generator tech.}

         DoomsDay_T:boolean;       {Does the player have the DoomsDay Weapon Tech.}
   {-=-=-=-=-=- Players Army Status -=-=-=-=-=-=-=-}
       {Soldier Storage Variables}
       {Human Type Soldiers}
         HumanN:longint;        {Number of normal human soldiers}
         GAHuman:longint;       {Number of GAHuman soldiers}
         SuperHuman:longint;    {Number of Super Human soldiers}
         HyperHuman:longint;    {Number of HyperHuman soldiers}
         MegaHuman:longint;     {Number of MegaHuman soldiers}

         BasicB_U:longint;      {Number of Basic BioBot Upgraded Soldiers}
         StandardB_U:longint;   {Number of Standard BioBot Upgraded Soldiers}
         AdvancedB_U:longint;   {Number of Advanced BioBot Upgraded Soldiers}
         AdvancedB2_U:longint;  {Number of Advanced-2 BioBot Upgraded Soldiers}
         BattleB_U:longint;     {Number of Battle BioBot Upgraded Soldiers}
         HeavyAttackB_U:longint;{Number of Heavy Attack BioBot upgraded Soldiers}
         OmegaB_U:longint;      {Number of Omega BioBot upgraded soldiers}
         OmegaBL2_U:Longint;    {Number of Omega L2 BioBot upgraded soldiers}

       {Ballistic Attack Weapons}
         AntiMatter_Q:longint;     {Number of Antimatter Missles The Player Has}
         FoodB_Q:longint;          {Number of Food Bombs The Player Has}
         NukeB_Q:longint;          {Number of nuclear missles the player has}
         RadiationB_Q:longint;     {Number of Radiation Bombs the player has}
         PopulationB_Q:longint;    {Number Of Population Bombs The Player Has}
         HypnoB_Q:longint;         {Number Of HypnoBombs the player has}

       {Player's Recon Drone}
         Recon1_Q:longint;     {Player starts out with this technology}
         Recon2_Q:longint;     {Did the player research Recon-2 Drone Tech.}
         Recon3_Q:longint;     {Did the player research Recon-3 Drone Tech.}

       {Player's War Vehicals}
         Tank_Q:longint;        {Did the player research tank tech.}
         HoverC_Q:longint;      {Did the player research Hover Craft Tech.}
         Jet_Q:longint;         {Did the player research Jet Tech.}

       {Misc / Advanced Buildings...and other variables for them}
         Miners_Guild:boolean;  {Does the player have the Miners guild?}
           NumMiners:longint;   {Number of miners that are unassigned to a mine}

         Fishing_Guild:boolean; {Does the player has the fishing guild?}
           NumFishers:longint;  {The amount of fishermen the player has}
           FishStock:longint;   {The amount of fish remaining}
         Construction_B:longint;{Amount of construction Sites the player has}
         Intell_B:boolean;      {Does the player have the Intelligence building}
           Num_Spy:longint;     {Number of spies the player has}
           Terrorists:longint;  {Number of terrorist the player has}
         Lottery_B:boolean;     {Does the player have the lottery set up}
      {---Global Defence Systems---}
        ODSnum:longint;         {How many defence satillites the player has}
        ODSNS:boolean;          {Is the player using the ODSNS?}
        GSGnum:longint;         {How many Global Shield Generators the player has}
        GroundTNum:longint;     {How many Ground Turrets The Player Has}
        AntiANum:longint;       {How many Anti-Aircraft Missle launchers Player has}
        DoomsDay_B:boolean;     {Does the player have DDay weapon Built}
   END; {End of the One_Player record Type}
        {Record Size For 1 One_player record on APRIL 28/96 = 427 bytes}

Var  {Global Variables}
  Play_F: file of One_Player;                    {File with all the players}
  Problem_file:text;

  A: Global_E;
  CurrentPlayer: one_player; {The current player}
  TempPlayer: One_Player; {Globaly Used To Store A Temporay Record}
  NewsFile:text;

{-=-=-=-=-=-=-=-=-=-=-=-=- Forwards For Procedures -=-=-=-=-=-=-=-=-=-=-=}
Procedure ReadSetup;
Procedure Welcome_To_Game; Forward;
Procedure Search_For_Player; Forward;
Procedure Create_New_Player; Forward;
Procedure Play_Game; Forward;
Procedure Write20Empires; Forward;
Procedure ExitGame; Forward;
Function ConvertDate:integer; Forward;
{-=-=-=-=-=-=-=-=-=-=END Of Forwards For Procedures -=-=-=-=-=-=-=-=-=-=-=}
Procedure ReadSetup;
{DOMINION Setup Program}
{this little program is going to get the new player variable values}
{from the text configuration file.}
Var
  Infile:text;
  A:Global_E;
  C:integer;

Function G:longint;
Var
  w:string;
  S:string;
  x:byte;
  we:longint;
Begin
  w:='';
  repeat
    readln(infile,s);
  until (s[1] <> '''') and (s <> '');
  if s[1]='~' then
    for x:=2 to length(s) do
      if not(s[x]='~') then
        w:=w+s[x]
      else
        break;
  val(w,we,c);
  g:=we;
end;

Function B:boolean;
Var
  w:string;
  S:string;
  x:byte;

Begin
  w:='';
  repeat
    readln(infile,s);
  until (s[1] <> '''') and (s <> '');
  if s[1]='~' then
    for x:=2 to length(s) do
      if not(s[x]='~') then
        w:=w+s[x]
      else
        break;
  if (w='F') or (w='f') then
    B:=False
  else B:=True;
end;

Begin
  assign(infile,'CONFIG.DOM');
  reset(infile);
  A.Set_Turns:=G;
  A.Set_Money:=G;
  A.Set_MoneyBank:=G;
  A.Set_Population:=G;
  A.Set_Food:=G;
  A.Set_Energy:=G;
  A.Set_HumanN:=G;
  A.Set_GaHuman:=G;
  A.Set_SuperHuman:=G;
  A.Set_HyperHuman:=G;
  A.Set_MegaHuman:=G;
  A.Set_BasicB_U:=G;
  A.Set_StandardB_U:=G;
  A.Set_AdvancedB_U:=G;
  A.Set_AdvancedB2_U:=G;
  A.Set_BattleB_U:=G;
  A.Set_HeavyAttackB_U:=G;
  A.Set_OmegaB_U:=G;
  A.Set_OmegaBL2_u:=G;
  A.Set_Foodb_Q:=G;
  A.Set_NukeB_Q:=G;
  A.Set_RadiationB_Q:=G;
  A.Set_PopulationB_Q:=G;
  A.Set_HypnoB_Q:=G;
  A.Set_AntiMatter_Q:=G;
  A.Set_Recon1_Q:=G;
  A.Set_Recon2_q:=G;
  A.Set_Recon3_Q:=G;
  A.Set_Tank_Q:=G;
  A.Set_HoverC_T:=G;
  A.Set_Jet_T:=G;
  A.Set_ODSnum:=G;
  A.Set_ODSNS:=B;
  A.Set_GSGnum:=G;
  A.Set_GroundTnum:=G;
  A.Set_AntiANum:=G;
  A.Set_DoomsDay_B:=B;
  A.Set_HydroT:=B;
  A.Set_WindT:=B;
  A.Set_SolarT:=B;
  A.Set_FossilT:=B;
  A.Set_FissionT:=B;
  A.Set_FusionT:=B;
  A.Set_CFusionT:=B;
  A.Set_HumanNT:=B;
  A.Set_GAhumanT:=b;
  A.Set_SuperHumanT:=b;
  A.Set_HyperHumanT:=b;
  A.Set_MegaHumanT:=b;
  A.Set_BasicBT:=b;
  A.Set_StandardBT:=b;
  A.Set_AdvancedBT:=b;
  A.Set_Advanced2BT:=B;
  A.Set_BattleBT:=b;
  A.Set_HeavyAttackBT:=b;
  A.Set_OmegaBT:=b;
  A.Set_OmegaL2BT:=B;
  A.Set_Recon1T:=B;
  A.Set_Recon2T:=B;
  A.Set_Recon3T:=B;
  A.Set_TankT:=B;
  A.SEt_HoverCT:=B;
  A.Set_JetT:=B;
  A.Set_NukeBT:=b;
  A.Set_RadiationBT:=b;
  A.set_HypnoT:=B;
  A.SEt_PopulationBT:=b;
  A.Set_FoodBT:=b;
  A.Set_AntiMatterT:=B;
  A.Set_ODST:=B;
  A.Set_ODSNST:=B;
  A.Set_GroundTT:=B;
  A.Set_AntiT:=B;
  A.Set_GSGT:=B;
  A.Set_DoomsDayT:=B;
  A.Set_Miners_Guild:=B;
  A.Set_NumMiners:=G;
  A.Set_Fishing_Guild:=B;
  A.Set_NumFishers:=G;
  A.Set_FishStock:=G;
  A.Set_Construction_B:=G;
  A.Set_Labs_B:=G;
  A.Set_Intell_B:=B;
  A.Set_Num_Spy:=G;
  A.Set_Terrorists:=G;
  A.Set_Lottery_B:=B;
  A.Set_GOLD:=G;
  A.Set_Silver:=G;
  A.Set_Iron:=G;
  A.Set_Copper:=G;
  A.Set_Nickel:=G;
  A.Set_Agricultural:=G;
  A.Set_Industrial:=G;
  A.Set_Desert:=G;
  A.Set_Urban:=G;
  A.Set_River:=G;
  A.Set_OceanSea:=G;
  A.Set_WasteLand:=G;
  A.Set_BuildingP:=G;
  A.Set_ResearchP:=G;
  Close(infile);
end;
Function CurrentDate:integer;
Var
  Yr,Mon,Da,Dow:word;
  StorDay:integer;
Begin
  getdate(yr,mon,da,dow);
  Case Mon of
            1: inc(storday,Da);            {jan}
            2: inc(storday,da+31);         {feb}
            3: inc(storday,da+31+28);      {mar}
            4: inc(storday,da+31+28+31);   {apr}
            5: inc(storday,da+31+28+31+30);{may}
            6: inc(storday,da+31+28+31+30+31); {june}
            7: inc(storday,da+31+28+31+30+31+30); {july}
            8: inc(storday,da+31+28+31+30+31+30+31); {august}
            9: inc(storday,da+31+28+31+30+31+30+31+31); {sept}
            10: inc(storday,da+31+28+31+30+31+30+31+31+30); {oct}
            11: inc(storday,da+31+28+31+30+31+30+31+31+30+31); {nov}
            12: inc(storday,da+31+28+31+30+31+30+31+31+30+31+31); {dec}
 end;
 CurrentDate:=Storday;
end;

Procedure Write20Empires;
Var
  DisplayPlayerRecord:longint;
  RE:One_Player; {Temp Record To Look For The Message Recieving Empire}
  CountLines:byte; {Used To Count 24 Lines and THen write <more> to the
                    screen}
  Chaf:char;
Begin
  set_FOREGROUND(11);
  Sclrscr;
  writeln(soutput,'Empires Number             Empire Name');
  writeln(soutput,'--------------             -----------');

  For DisplayPlayerRecord:= 0 to filesize(play_F) do
    Begin
      Seek(Play_F,DisplayPlayerREcord);
      Read(Play_F,RE);
      set_Foreground(13);
      sgoto_xy(6,countlines+1);
      Set_Foreground(10);
      write(SOUTPUT, DisplayPlayerRecord);
      Sgoto_xy(48,countlines+1);
      inc(CountLines);
        if countlines=20 then begin
           SRead_Char(Chaf);
           CountLines:=0;
         end;
    end;
end;

Procedure Messages(ControlCode:byte);
{Control Codes
0: Enter New Message
1: Search For Players Messages}
Var
  MessageFile:File Of Message_Type;
  ONeWord:string;
  OnWord:boolean;
  Inchar:char;
  CurrentMessage:Message_Type;
  CountChar:byte;
  CountLn:byte;

Begin
  assign(MessageFile,Message_Outfile_Name);
  {$I-}
   Reset(MessageFile);
  {$I+}
  if ioresult <> 0 then
    rewrite(MessageFile);

  If ControlCode=0 then
    Begin
      {Displayfile('COMMSCR.ANS');}
      Write20Empires;
end;
end;
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
Procedure FadeWrite(output:string; locx:byte; locy:byte; d:byte; Color:byte);
Var
  x,y:byte;
Begin
  output:=output+'  ';
  for x:= 1 to length(output) do
    Begin
      Set_Foreground(15);
      Swritexy(locx+x,locy,output[x]);
      delay(d);

      Set_Foreground(Color);
      LowVideo;
      if x >=2 then
        Swritexy(locx+x-1,locy,output[x-1])
      else
        Swritexy(locx+x,locy,output[x]);

      delay(d);

      Set_Foreground(Color);
      HighVideo;
      if x >=3 then
        SWRITEXY(locx+x-2,locy,output[x-2])
      else
        SWRITEXY(locx+x,locy,output[x]);
      delay(d);
  end;
end;

{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
Procedure Search_For_Player;
Var
  CurrentPos:longint; {The current search position}
  StatusOkay:boolean;
  testchar:char;
Begin
 reset(Play_F);
 if filesize(Play_F) =0 then
     Create_New_Player
 else
    for CurrentPos:= 0 to filesize(Play_F) do
      begin
        seek(Play_F,Currentpos);
        read(Play_F,CurrentPlayer);
        if (Currentplayer.Players_1stName=User_First_Name) and
           (Currentplayer.Players_LstName=User_Last_Name) then
           break
        else
         Create_New_Player;
       end;
end;
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}

Procedure Create_New_Player;
Var
  Empire_N:string;
  World_N:string;

  Temp_Str1:string[20];
  Temp_Str2:string[20];

  x,y:byte;
  okay:boolean;
  Search_Records:boolean;
  RecordTrac:longint;

begin
  SCLRSCR;
  okay:=false;
  FadeWrite('Welcome To Dominion.',1,1,10,10);
  swriteln('');
  Set_ForeGround(13);
  FadeWrite('Name Your World <?>: ',1,2,10,15);
  Set_Foreground(11);
  RecordTrac:=0;
  Okay:=False;

  REPEAT
    if World_N[1]=#3 Then  {The Chat Thing..Should Implement this somehow}
    Begin
      SCLRSCR;
      okay:=false;
      FadeWrite('Welcome To Dominion.',1,1,10,10);
      swriteln('');
      Set_ForeGround(13);
      FadeWrite('Name Your World <?>: ',1,2,10,15);
      Set_Foreground(11);
    repeat
      SGoto_xy(23,2);
      sclreol;
      Prompt(World_N,20,False);
    until World_n<>'';
    end
    else

    repeat
      SGoto_xy(23,2);
      sclreol;
      Prompt(World_N,20,False);
    until World_n<>'';

    if World_N[1]='?' Then {If the Player Wants Help, run HELP menu code X}
      Begin
        Okay:=False;
        Set_ForeGround(14);
        swriteln('Enter The Name You Want For Your World.');
        Set_ForeGround(11);
        Search_Records:=False;
      end
    else
      Begin
        Temp_Str1:=World_n; {Makes Temp_Str1 equal to Worlds_N}
        Search_Records:=True;
      end;
    {--------------------------------}
    For x:=0 To Length(temp_Str1) do
      Begin
        If Temp_Str1[x]=#32 then
          begin                           {This section erases all the spaces}
            delete(Temp_Str1,x,1);        {and converts Temp_Str1 To Upper Case}
            dec(x);
          end;
        Temp_Str1[x]:=Upcase(Temp_Str1[x]);
      end;
     {--------------------------------}

    Swriteln('Checking For Duplicate World Names. Please Wait...');
    Currentplayer.Worlds_NameC:=Temp_Str1; {Sets The Search String For The}
                                           {Player}

     if Filesize(Play_F)=0 then  {If There Are no other player records then}
         break;                  {break out of REPEAT UNTIL loop get}
                                 {Empire Name}

   {-----------BEGINING OF SEARCH ENGINE----------------------------------}
     {This part searches the PLAYERS.DAT File for duplicate World Names...}
     {This small little search Engine was a total pain in the ass! It better}
     {Keep Working.}
   if search_Records=true then
     For RecordTrac:=0 to Filesize(Play_F) do
       Begin
         Seek(PLAY_F,RecordTrac);
         read(Play_F,TempPlayer);

         If TempPlayer.Worlds_NameC=Currentplayer.Worlds_NameC then
          Begin
            Okay:=False;
            Set_ForeGround(10);
            Swriteln('Duplicate World Name Found.');
            Swriteln('Please Use Different Name.');
            Set_ForeGround(11);
            Break;
          end;

         If EOF(play_f) then
           Begin
             Okay:=True;
             Break;
           end;
     end;
   {-----------END OF SEARCH ENGINE---------------------------------------}
  UNTIL Okay=True;
  Search_Records:=False;
  okay:=false;
  {This Part Get's the new players Empire Name}
  swriteln('');
  Currentplayer.Worlds_Name:=World_n;
  fadewrite('New World',wherex,6,10,10);
  fadewrite(Currentplayer.Worlds_name,wherex,6,10,15);
  fadewrite('Discovered.',wherex,6,10,10);
  swriteln('');
  FadeWrite('Name Your Empire <?>: ',wherex,wherey,10,15);
  Set_Foreground(11);
  x:=wherex;
  y:=wherey;
  RecordTrac:=0;

  REPEAT

   if Empire_N[1]=#3 Then  {The Chat Thing..Should Implement this somehow}
    Begin
      SCLRSCR;
      okay:=false;
      FadeWrite('Name Your Empire <?>: ',1,10,10,15);
      Set_Foreground(11);
      repeat
        SGoto_xy(x,y);
        sclreol;
        Prompt(Empire_N,20,False);
      until Empire_N<>'';
     end
    else
    repeat
      SGoto_xy(x,y);
      sclreol;
      Prompt(Empire_N,20,False);
    until Empire_n<>'';

    if Empire_N[1]='?' Then {If the Player Wants Help, run HELP menu code X}
      Begin
        Okay:=False;
        Set_Foreground(14);
        swriteln('Enter The Name You Want For Your Empire.');
        Set_Foreground(11);
        Search_Records:=False;
      end
    else
      Begin
        Temp_Str1:=Empire_n; {Makes Temp_Str1 equal to Worlds_N}
        Search_Records:=True;
      end;
    {--------------------------------}
    For x:=0 To Length(temp_Str1) do
      Begin
        If Temp_Str1[x]=#32 then
          begin                           {This section erases all the spaces}
            delete(Temp_Str1,x,1);        {and converts Temp_Str1 To Upper Case}
            dec(x);
          end;
        Temp_Str1[x]:=Upcase(Temp_Str1[x]);
      end;
     {--------------------------------}
    CurrentPlayer.Empires_NameC:=Temp_Str1;
    Swriteln('Checking For Duplicate Empire Names. Please Wait...');

   {-----------BEGINING OF SEARCH ENGINE----------------------------------}
     {This part searches the PLAYERS.DAT File for duplicate World Names...}
     {This small little search Engine was a total pain in the ass! It better}
     {Keep Working.}
     if Filesize(Play_F)=0 then {if there are no records then}
         break;                 {The Empire Name Is Okay}

     For RecordTrac:=0 to Filesize(Play_F) do
       Begin
         Seek(PLAY_F,RecordTrac);
         read(Play_F,TempPlayer);

         If TempPlayer.Empires_NameC=Currentplayer.Empires_NameC then
          Begin
            Okay:=False;
            Set_ForeGround(10);
            Swriteln('Duplicate Empire Name Found.');
            Swriteln('Enter A New Empire Name');
            Set_ForeGround(11);
            Break;
          end;

         If EOF(play_f) then
           Begin
             Okay:=True;
             Break;
           end;
     end;
   {-----------END OF SEARCH ENGINE---------------------------------------}
  UNTIL Okay=True;

{This part is run after the search for the players name...}
{So this procedure is dependant on procedure ENTER_GAME to create}
{A New TRACK_PLAYER_RECORD file when it can't find one}
  Reset(Play_F);
  {This part sets all the information for the new player}
  currentplayer.Players_1stName:=USER_FIRST_NAME; {Players first name}
  currentplayer.Players_LstName:=USER_LAST_NAME; {Players last name}
  currentplayer.money:=A.Set_Money;  {Sets Money}
  currentplayer.Building_P:=A.Set_BuildingP; {Sets Building pionts}
  currentplayer.Research_P:=A.Set_ResearchP; {Sets Research Points}
  currentplayer.population:=A.Set_population; {Players Population}
  {Sets Players Technology}
  currentplayer.HumanN_T:=true; {Normal Human Soldier Technology}
  currentplayer.Fossil_T:=true; {Fossil Fuel Technology}
  Currentplayer.recon1_t:=true; {Level 1 Recon Drone Technology}

  {Sets Regions}

  {Setting Ready Costs}

  currentplayer.agricult.Ready_Reg:=round((1000+(1000*(0.0001 * currentplayer.agricult.quantity))));
  Currentplayer.Industrial.Ready_Reg:=round((1700+(1700*(0.0001 * currentplayer.industrial.quantity))));
  Currentplayer.Desert.Ready_Reg:=round((500+(500*(0.0001 * Currentplayer.desert.quantity))));
  Currentplayer.Urban.Ready_Reg:=round((1500+(1500*(0.0001 * Currentplayer.urban.quantity))));
  Currentplayer.Volcanic.Ready_Reg:=Round((1300+(1300*(0.0001 * Currentplayer.volcanic.quantity))));
  Currentplayer.River.Ready_Reg:=Round((1300+(1300*(0.0001 * Currentplayer.River.quantity))));
  Currentplayer.OceanSea.Ready_Reg:=Round((1000+(1000*(0.0001 * Currentplayer.OceanSea.quantity))));
  Currentplayer.Wastelands:=0;

  seek(Play_F,filesize(Play_F));
  write(Play_F,Currentplayer); {Writes The New Player To The Players.Dat File}

end;
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
Procedure Welcome_To_Game;
Var
 UserInput:char; {Uses To Get Input From The User}
Begin
  DISPLAYFILE('c:\TP\GAME\MEGAGAME\SCREENS\SCREEN1.ANS');
  fadewrite('Press <SPACE> To Enter DOMINION',23,20,15,13);
  repeat
    SRead_Char(USERINPUT);
  until USERINPUT in[#32,#13];
  SCLRSCR;
  Set_FOreGround(13);
  writeln(soutput);
  write(Soutput,'1. ');
  Set_Foreground(10);
  writeln(soutput,'Enter Dominion');
  writeln(soutput);
  Set_ForeGround(13);
  write(Soutput,'2. ');
  Set_ForeGround(5);
  writeln(Soutput,'Instructions For DOMINION');
  writeln(soutput);
  Set_Foreground(13);
  write(Soutput,'3. ');
  Set_Foreground(5);
  writeln(soutput,'Empire Rankings');
  writeln(soutput);
  Set_ForeGround(13);
  write(soutput,'4. ');
  Set_Foreground(5);
  writeln(soutput,'Galactic News');
  writeln(soutput);
  Set_ForeGround(13);
  write(soutput,'5. ');
  set_Foreground(5);
  writeln(soutput,'DOMINION Story Line');
  writeln(soutput);
  Set_Foreground(13);
  write(soutput,'6. ');
  Set_Foreground(7);
  writeln(soutput,'Quit Game');
  writeln(soutput,'');
  FadeWrite('Enter Your Choice: ',1,wherey,15,10);
  {Show Main Welcome Screen Here: Screen0}
  {Show Main Menu: Screen1}
    {Main Sc reen Options}
    {(E)nter Game}
    {(I)nstructions}
    {(R)ankings}
    {(G)alatic News}
    {(S)tory Line}
    {(Q)uit Game}
  repeat
    SREAD_CHAR(UserInput);
  until UPCASE(UserInput) in['1','2','3','4','5','6',#3]; {Does that until ...}
  write(userinput);
  case upcase(UserInput) of

    '1': Search_For_Player;

    '2': Begin
           moreok:=true;
           DISPLAYFILE('C:\tp\game\megagame\screens\INSTRUCT.ANS');
           Welcome_To_Game;
         end;

    '5': Begin
           MOREOK:=true;
           DISPLAYFILE('C:\tp\game\megagame\screens\STORYLN.ANS');
           Welcome_To_Game;
         end;

    '6': ExitGame;

    'G': Displayfile('C:\TP\GAME\MEGAGAME\NEWSFILE.ANS');

     #3: Welcome_To_Game;

  end;
end;
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}
Procedure ExitGame;
Begin
  Set_Foreground(14);
  Swriteln('Returning To: '+ Board_Name);
  sWritelN('Thank Your For Playing Dominion By Benson Wong');
  if registered=true then
    swriteln('Dominion Is Registered To ' + sysop_First_Name + ' ' + Sysop_Last_Name + ' of The ' + Board_Name)
  else Swriteln('Dominion Is Not Registered. Please Ask Your Sysop(s) To Register.');
  Swriteln('');
  Halt;
End;
{=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-}

Procedure Play_Game;
  {Main Menu Commands:
    (A)ttack Menu
    (M)aintence Menu


  }


begin
end;
Begin
  ReadSetup;
  initdoordriver('GAME.CTL'); {initiates the door driver}
  Assign(NewsFile,newsFileName);
  Assign(Play_f,Player_Data_filename);
  Assign(Problem_File,Problem_log_Filename);
  {$I-}
    Reset(NewsFile);        {This is where the news file is opened}
  {$I+}
    if ioresult <> 0 then
      rewrite(NewsFile);
  {$I-}
    Reset(Play_F);
  {$I+}
  if ioresult <> 0 then      {This is where the Players.Dat is open or created}
    Rewrite(Play_F);
  Close(Play_F);
  {$I-}
    reset(Problem_FIle);
  {$I+}
    if ioresult <> 0 then
      rewrite(Problem_File);

  writeln('BetaVersion .01b By I-Caris Daemonica');
  Welcome_To_game;
  close(Problem_File);
  Close(Play_F);
end.