// Dati di esempio SOLO per l'anteprima (nell'app vera arrivano dal server e da Maxroll)
(() => {
const T = {WW:'Waygate',SS:'Stronghold',ZZ:'ZoneName',WWB:'World Boss',BBL:'Boss Lair',DD:'Dungeon',AAoL:'Altar of Lilith',TToA:'Tenet of Akarat'};
const Z = {FP:'Fractured Peaks',S:'Scosglen',DS:'Dry Steppes',K:'Kehjistan',H:'Hawezar',N:'Nahantu'};
const M = [[29,"SS","Nostrava","FP",41,58],[30,"SS","Malnok","FP",46,108.5],[31,"SS","Kor Dragan","FP",61,76.5],[32,"WW","Nevesk","FP",30,60],[33,"WW","Margrave","FP",30,79],[34,"WW","Yelesna","FP",28,102.5],[35,"WW","Bear Tribe Refuge","FP",52.5,110],[36,"WW","Kyovashad","FP",45.5,82],[38,"WW","Menestad","FP",56,69],[42,"DD","Light's Watch","FP",36.5,79],[45,"DD","Immortal Emanation","FP",43,98.5],[53,"DD","Black Asylum","FP",52,96],[3,"AAoL","Altar of Lilith","FP",31,63],[6,"AAoL","Altar of Lilith","FP",42,73],[21,"AAoL","Altar of Lilith","FP",51.5,89],[71,"WW","Under the Fat Goose Inn","S",68.5,90],[72,"WW","Tirmair","S",72,95],[87,"WW","Corbach","S",75.5,79.5],[100,"WW","Cerrigar","S",72,56],[102,"WW","Braestaig","S",77.5,55],[111,"WW","Marowen","S",80,29],[119,"WW","Tur Dulra","S",75.5,19.5],[122,"WW","Firebreak Manor","S",68.75,46],[84,"DD","Garan Hold","S",75,88],[96,"DD","Sarat's Lair","S",66.5,65.5],[130,"WW","Farobru","DS",68.5,16],[139,"WW","Fate's Retreat","DS",58,29],[143,"WW","Hidden Overlook","DS",47.75,29.5],[147,"WW","Ruins of Qara-Yisu","DS",30,28],[160,"WW","The Onyx Watchtower","DS",57,6],[164,"WW","Ked Bardu","DS",65,-5],[186,"WW","Alzuuda","DS",42.5,-35],[190,"WW","Jirandai","DS",24,-6],[195,"WW","Altar of Ruin","K",23.5,-35],[203,"WW","Imperial Library","K",7,-33],[205,"WW","Iron Wolves Encampment","K",5.5,-43],[224,"WW","Gea Kul","K",-27,-58],[232,"WW","Denshar","K",-20,-22],[245,"WW","Tarsarak","K",-1.5,12],[253,"WW","Zarbinzet","H",-22,33],[267,"WW","Vyeresz","H",-34,73],[281,"WW","Backwater","H",-33.5,115],[295,"WW","Ruins of Rakhat Keep","H",-8,74],[302,"WW","Wejinhani","H",11,63.5],[311,"WW","The Tree of Whispers","H",15,108],[395,"ZZ","Scosglen","S",80,9],[396,"ZZ","Fractured Peaks","FP",63,134],[397,"ZZ","Hawezar","H",-32,130],[398,"ZZ","Kehjistan","K",16,-71],[399,"ZZ","Dry Steppes","DS",65,-40],[504,"SS","Moordain Lodge","S",76.5,101],[505,"SS","Tur Durla","S",76,18],[506,"SS","Hope's Light","S",83.5,40],[507,"SS","The Onyx Watchtower","DS",55.5,4],[509,"SS","Temple of Rot","DS",38.5,-11],[510,"SS","Altar of Ruin","K",25,-37],[511,"SS","Alcarnus","K",11,-5],[512,"SS","Omath's Redoubt","K",-29,-29],[513,"SS","Crusader's Monument","H",3,36],[515,"SS","Eriman's Pyre","H",18,90],[572,"WWB","Hawezar","H",-33,44],[573,"WWB","Kehjistan","K",20,9],[574,"WWB","Fractured Peaks","FP",40,131],[575,"WWB","Scosglen","S",77.5,77.5],[576,"WWB","Dry Steppes","DS",38,40],[627,"BBL","Hall of the Penitent (Grigoire)","DS",56,-6.5],[628,"BBL","Gaping Crevasse (Duriel)","K",-37,-38.5],[629,"BBL","Malignant Burrow (Varshan)","H",14.5,111.5],[630,"BBL","Glacial Fissure (The Beast in the Ice)","FP",38,76],[631,"BBL","Ancient's Seat (Lord Zir)","FP",39,111],[696,"BBL","Hanged Man's Hall (Andariel)","K",-1,22.5],[720,"WW","Kichuk","N",-37.25,24],[721,"WW","Ichorfall","N",-64.5,5.5],[722,"WW","Gates of the Necropolis","N",-72.5,-18.5],[723,"WW","Kurast Bazaar","N",-61,-47],[724,"WW","Samuk","N",-68,-81.75],[725,"WW","Kurast Docks","N",-39.5,-65.5],[729,"WWB","Nahantu","N",-53.5,22.25],[760,"BBL","Fields of Judgement (Urivar)","N",-37.14,12.24],[761,"BBL","Harbinger's Den (Harbinger of Hatred)","N",-42.08,-71.28],[730,"TToA","Tenet of Akarat","N",-37.5,13],[735,"TToA","Tenet of Akarat","N",-55.5,11]];
const markers = M.map(([id,t,n,z,a,b]) => ({id, type:T[t], name:n, zone:Z[z], coords:[a,b]}));
const now = Math.floor(Date.now()/1000);
const H = 3600, wb0 = Math.ceil((now - 30*60) / 1800) * 1800 + 25*60;
const events = [];
const bosses = [['Azmodan',[{name:'Scosglen'},{name:'Skovos'}]],['Wandering Death',[{name:'Dry Steppes'}]],['Avarice',[{name:'Fractured Peaks'},{name:'Nahantu'}]],['Ashava',[{name:'Kehjistan'}]]];
bosses.forEach((b,i) => events.push({kind:'boss', start:wb0 + i*210*60, end:wb0 + i*210*60 + 900, title:b[0], zones:b[1]}));
const h0 = now - now % H; for (let t = h0; t < now + 12*H; t += H) events.push({kind:'helltide', start:t, end:t+55*60, title:'Marea infernale'});
const l0 = now - (now % 1500) + 600; for (let t = l0; t < now + 12*H; t += 1500) events.push({kind:'legion', start:t, end:t+240, title:'Raduno della Legione'});
events.sort((a,b)=>a.start-b.start);
const rob16 = [['S','Ball Lighting Sorc','150','sorcerer'],['S','Firewall Sorc','148','sorcerer'],['S','Abyss Dread Claw Warlock','140 3m now permaish Demonform','warlock'],['S','Leap Druid POG','141 but x3 DMG buff + Grizzly Rage','druid'],['S','Poison DoK Flurry Rogue','not nerfed still crazy','rogue'],['S','Flay Barb','150 DPS Wise its crazy but low AoE','barbarian'],['S','Cpt. America Paladin','Still solid 147+','paladin'],['A','Whirlwind Barb','147 in 9m hard push full glass canon','barbarian'],['A','Frenzy Barb','145 11m','barbarian'],['A','Double Swing Barb','50% x dmg + Set Buffs','barbarian'],['A','Bone Spirit Necro','EZ 143 on PTR','necromancer'],['A','Fire Ferocity Spiritborn','140 EZ on PTR','spiritborn'],['B','Minion Barb','Infinite Ancient bug finally fixed','barbarian']].map((r,i)=>({tier:r[0],tierNote:'',build:r[1],info:r[2],url:'https://d4builds.gg/builds/x'+i+'/',cls:r[3],rank:i+1}));
const lv16 = [['S','Stinger Spiritborn','Toxic Skin + Stinger OP BEST LvL can do sub 1h','spiritborn'],['S','Incinerate Hydra Sorc','Fire Sorcs got even better now!','sorcerer'],['S','Whirlwind Barb','Ramping at Lvl 15 with Tornado - perfect Endgame transition','barbarian'],['A','Blazing Scream Warlock','Perfect transition into Endgame','warlock']].map((r,i)=>({tier:r[0],build:r[1],info:r[2],url:'https://d4builds.gg/builds/'+r[1].toLowerCase().replace(/ /g,'-')+'-leveling/',cls:r[3],rank:i+1}));
const builds = [
 {src:'Maxroll',title:'Whirlwind Barbarian Endgame Build Guide',cls:'barbarian',type:'Endgame',author:'aoinomiku',url:'https://maxroll.gg/d4/build-guides/whirlwind-barbarian-guide',profile:'qeyb7t0x',updated:now-6*86400,tier:'A',info:'147 in 9m hard push full glass canon'},
 {src:'Maxroll',title:'Whirlwind Barbarian Leveling Guide',cls:'barbarian',type:'Leveling',author:'aoinomiku',url:'https://maxroll.gg/d4/build-guides/whirlwind-barbarian-leveling-guide',profile:'qeyb7t0x',updated:now-6*86400,tier:'S',info:'Ramping at Lvl 15 with Tornado'},
 {src:'D4Builds',title:'Flay Barbarian',cls:'barbarian',type:'Endgame',url:'https://d4builds.gg/builds/flay-barbarian-endgame/',author:'Rob2628',tier:'S',info:'150 DPS Wise its crazy but low AoE'},
 {src:'Maxroll',title:'Upheaval Barbarian Endgame Build Guide',cls:'barbarian',type:'Endgame',author:'Maxroll',url:'https://maxroll.gg/d4/build-guides/upheaval-barbarian-guide',profile:'',updated:now-20*86400},
 {src:'D4Builds',title:'Frenzy Barbarian',cls:'barbarian',type:'Endgame',url:'https://d4builds.gg/builds/frenzy-barbarian-endgame',author:'Rob2628',tier:'A',info:'145 11m'},
 {src:'Mobalytics',title:'HotA Barbarian - Endgame Pushing',cls:'barbarian',type:'Endgame',author:'NeverMind',url:'https://mobalytics.gg/diablo-4/builds',likes:1840,verified:true},
 {src:'Maxroll',title:'Earthquake Barbarian Endgame Build Guide',cls:'barbarian',type:'Endgame',author:'Maxroll',url:'https://maxroll.gg/d4/build-guides/earthquake-barbarian-guide',profile:''},
 {src:'Mobalytics',title:'Double Swing Barbarian',cls:'barbarian',type:'Endgame',author:'M1PY',url:'https://mobalytics.gg/diablo-4/builds',likes:960,tier:'A',info:'50% x dmg + Set Buffs'},
];
const planner = {id:'qeyb7t0x',name:'Whirlwind Endgame Barbarian',class:'barbarian',url:'https://maxroll.gg/d4/planner/qeyb7t0x',
 variants:[
  {name:'Leveling 1-70',skills:[['Affondo','Lunging Strike'],['Turbine','Whirlwind'],['Pestone','Ground Stomp'],['Urlo di sfida','Challenging Shout'],['Grido di guerra','War Cry'],['Richiamo degli Antichi','Call of the Ancients']].map(([n,e],i)=>({key:['Barbarian_LungingStrike','Barbarian_Whirlwind'][i]||'x',name:n,en:e})),gear:[]},
  {name:'Starter',skills:[],gear:[]},{name:'Midgame',skills:[],gear:[]},
  {name:'Endgame',skills:[['Affondo','Lunging Strike'],['Turbine','Whirlwind'],['Pestone','Ground Stomp'],['Urlo di sfida','Challenging Shout'],['Grido di guerra','War Cry'],['Richiamo degli Antichi','Call of the Ancients']].map(([n,e])=>({key:'x',name:n,en:e})),
   gear:[
    {slot:4,type:'Elmo',typeEn:'Helm',name:'Elmo con Zanne di Joritz il Poderoso',en:"Tuskhelm of Joritz the Mighty",kind:'unique',sockets:['Rune di condizione|Condition rune','Rune di effetto|Effect rune']},
    {slot:5,type:'Protezione per torso',typeEn:'Chest Armor',name:'Enigma',en:'Enigma',kind:'runeword',sockets:[]},
    {slot:8,type:'Mazza a due mani',typeEn:'Two-Handed Mace',name:'Infinità',en:'Infinity',kind:'runeword'},
    {slot:9,type:'Arma ad asta',typeEn:'Polearm',name:'Intuizione',en:'Insight',kind:'runeword'},
    {slot:11,type:'Spada',typeEn:'Sword',name:'Mlor della Furia di Sescheron',en:"Sescheron's Fury",kind:'unique'},
    {slot:12,type:'Spada',typeEn:'Sword',name:'Aspetto dei Colpi Intensi',en:'Heavy Hitting Aspect',kind:'legendary'},
    {slot:13,type:'Guanti',typeEn:'Gloves',name:'Presa Devastante di Gohr',en:"Gohr's Devastating Grips",kind:'unique'},
    {slot:14,type:'Pantaloni',typeEn:'Pants',name:'Tasse dell\'Alba',en:'Tassets of the Dawning Sky',kind:'unique'},
    {slot:15,type:'Stivali',typeEn:'Boots',name:'Aspetto dell\'Abbraccio dell\'Anima',en:'Soul Embrace Aspect',kind:'legendary'},
    {slot:16,type:'Anello',typeEn:'Ring',name:'Aspetto dell\'Attrito Sprezzante',en:'Defiant Attrition Aspect',kind:'legendary'},
    {slot:17,type:'Anello',typeEn:'Ring',name:'Aspetto del Peccato Cupo',en:'Grim Sin Aspect',kind:'legendary'},
    {slot:18,type:'Amuleto',typeEn:'Amulet',name:'Melted Heart of Selig',en:'Melted Heart of Selig',kind:'mythic'},
    {slot:21,type:'Talismano',typeEn:'Charm',name:'Onere Spettrale',en:'Wraith Onus',kind:'talisman'},
   ]},
 ],
 filters:[{name:'Whirlwind Leveling',code:'CjoKFkJhc2VzIGZvciBDdWJlIFVwZ3JhZGUQAh3+gez/IhEI'},{name:'Whirlwind Starter/Midgame',code:'ChwKC0FsbCBNeXRoaWNzEAAdAAD//yIECAEgICgBCh8K'}]};
planner.variants[1] = {...planner.variants[3], name:'Starter'}; planner.variants[2] = {...planner.variants[3], name:'Midgame'};
const lootBosses = [
 {id:'varshan',name:'Echo of Varshan',location:'Malignant Burrow - Tree of Whispers',kind:'boss',items:[0,1,2,3,4],summon:['Chiave del covo','Kry','Thul','Tam'],info:['Echo of Varshan’s summoning location, Malignant Burrow, is beside the Tree of Whispers. The waypoint itself is found in Hawezar.','Material Required: 1 Lair Key.','Damage Type: Shadow and Physical']},
 {id:'duriel',name:'Duriel, King of Maggots',location:'Gaping Crevasse - Kehjistan',kind:'boss',items:[5,6,7],summon:['Chiave del covo']},
 {id:'zir',name:'Lord Zir',location:"Ancient's Seat - Fractured Peaks",kind:'boss',items:[8,9]},
 {id:'belial',name:'Belial, Lord of Lies',location:'Kehjistan',kind:'boss',items:[10],info:["Summoning Material Required: 2 Betrayer's Husk",'Damage Type: Shadow and Poison.']},
 {id:'mythic',name:'Mitici (tutti i boss, raro)',kind:'mythic',items:[11,12]},
 {id:'random',name:'Nessun boss: drop casuale o Cubo Horadrico',kind:'special',items:[13]},
];
const L = (name,en,type,srcs,o={}) => ({name,en,type,sources:srcs.map(([b,c])=>({boss:b,cls:c||'',kind:'boss'})),...o});
const items = [
 L('Frammento di Verathiel',"Shard of Verathiel",'Spada',[['varshan']]),
 L('Centomila passi','100,000 Steps','Stivali',[['varshan','barbarian']],{classes:['barbarian']}),
 L('Eccesso','Overkill','Mazza a due mani',[['varshan','barbarian']],{classes:['barbarian']}),
 L('Corona Autunnale','Autumnal Crown','Elmo',[['varshan','druid']],{classes:['druid']}),
 L('La Mano di Naz','The Hand of Naz','Guanti',[['varshan','necromancer']],{classes:['necromancer']}),
 L('Presa Devastante di Gohr',"Gohr's Devastating Grips",'Guanti',[['duriel','barbarian']],{classes:['barbarian']}),
 L('Potenza di Qual-Kehk','Might of Qual-Kehk','Elmo',[['duriel','barbarian']],{classes:['barbarian']}),
 L('Il Basilisco','The Basilisk','Bastone',[['duriel','druid']],{classes:['druid']}),
 L('Elmo con Zanne di Joritz il Poderoso','Tuskhelm of Joritz the Mighty','Elmo',[['zir','barbarian']],{classes:['barbarian']}),
 L('Arcadia','Arcadia','Arma',[['zir']]),
 L('Mazza di Re Leoric','Mace of King Leoric','Mazza',[['belial']]),
 L('Visaggio di Andariel',"Andariel's Visage",'Elmo',[['mythic']],{mythic:true}),
 L('Cuore Fuso di Selig','Melted Heart of Selig','Amuleto',[['mythic']],{mythic:true}),
 L('Pugni del Fato','Fists of Fate','Guanti',[['random']]),
];
const patches = {articles:[{title:'Diablo IV Patch Notes (3.3)',url:'https://news.blizzard.com/en-us/article/24316980/diablo-iv-patch-notes',patches:[{title:'3.3.0 Build #74500 (All Platforms)—December 1, 2026',version:'3.3.0',date:now+5*86400,html:'<h2>Season of Tyrael\'s Gambit</h2><p>Season 16 begins with new seasonal powers and a revamped Tower.</p><h3>Barbarian</h3><ul><li>Whirlwind damage increased by 20%.</li><li>Flay now applies Vulnerable for 2 seconds.</li></ul><h3>Sorcerer</h3><ul><li>Ball Lightning projectile count increased.</li></ul>'}]},{title:'Diablo IV Patch Notes (3.2)',url:'https://news.blizzard.com/en-us/article/24295395/diablo-iv-patch-notes-3-2',patches:[{title:'3.2.4 Build #74043 (All Platforms)—November 17, 2026',version:'3.2.4',date:now-8*86400,html:'<h2>Bug Fixes</h2><ul><li>Fixed an issue where socketed Runes or Gems would be lost when Recycling a Unique Item with the Horadric Cube.</li><li>Fixed an issue where Bloodwyrms spawned due to the Hellmouth War Plan modifier could deal massive damage.</li></ul>'}]}],forum:[{title:'Known Issues - Season 15',excerpt:'We are aware of an issue where some players may be disconnected when entering the Tower.',at:now-86400,author:'PezRadar',url:'https://us.forums.blizzard.com/en/d4/'},{title:'Hotfix 3.2.4a',excerpt:'A hotfix has been deployed to address several issues.',at:now-2*86400,author:'Blizzard',url:'https://us.forums.blizzard.com/en/d4/',html:'<ul><li>Fixed a crash when opening the Horadric Cube.</li><li>Mother\'s Blessing will now correctly apply to Infernal Hordes.</li></ul>'}]};
let macros = [];
const state = {version:'2026.11.25 · anteprima',frameless:false,season:16,seasons:[{n:16,ptr:true,live:false},{n:15,ptr:false,live:true}],settings:{lang:'it',notifications:true,notifyBoss:true,notifyBefore:5,blur:true,motion:true,accent:'ember',remoteUrl:'https://www.aversano.design/hr-data/',remoteToken:'••••',favorites:[]},remote:{configured:true,error:'',at:now-120,errors:{}},gameData:'',d4p:{path:'C:\\Users\\avers\\Desktop\\Utility\\D4Pressure.exe',running:false},classesIt:{barbarian:'Barbaro',druid:'Druido',necromancer:'Negromante',rogue:'Tagliagole',sorcerer:'Incantatore',spiritborn:'Spiritico',paladin:'Paladino',warlock:'Stregone'}};
window.DEMO = (path, opts) => {
  const p = path.split('?')[0];
  if (p === '/api/state') return state;
  if (p === '/api/events') return {now, events, bossSource:'helltides.com'};
  if (p === '/api/builds') return {season:16, builds, rob:{sheet:'S16 Tier List WIP', tier:rob16, leveling:lv16, wip:true}};
  if (p === '/api/build') return planner;
  if (p === '/api/loot') return {season:16, items, bosses:lootBosses, sources:['d4builds.gg (PTR)','maxroll.gg','mobalytics.gg'], ptr:true};
  if (p === '/api/map') return {result:{pageContext:{markers}}};
  if (p === '/api/mapfound') return {};
  if (p === '/api/patches') return patches;
  if (p === '/api/macro') { if (opts?.method === 'POST') { const b = JSON.parse(opts.body); if (path.includes('delete')) { macros = macros.filter(x => x.id !== b.id); return macros; } b.id = b.id || String(Date.now()); macros = macros.filter(x => x.id !== b.id).concat([b]); return b; } return macros; }
  if (p === '/api/settings') { if (opts?.body) Object.assign(state.settings, JSON.parse(opts.body)); return state.settings; }
  if (p === '/api/macro/launch') return {ok:true};
  if (p === '/api/refresh') return state.remote;
  return undefined;
};
})();
