Unit ReadSetup;
{DOMINION Setup Program}
{this little program is going to get the new player variable values}
{from the text configuration file.}
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

         Set_Construction_B:longint;{Number of war construction factories the player has}
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
  Close(infile);
end.
