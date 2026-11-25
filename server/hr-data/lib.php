<?php
// Hellrise – raccolta dati lato server (hosting PHP, es. Aruba)
// Scarica in sottofondo build (Maxroll, D4Builds, Mobalytics), la tier list di Rob2628,
// le tabelle "chi droppa cosa" e gli orari dei boss del mondo, e li tiene divisi per stagione
// (solo la stagione precedente e l'ultima).

// stesso codice segreto di BetterStomp se config.php qui non c'è
if (is_file(__DIR__ . '/config.php')) require __DIR__ . '/config.php';
elseif (is_file(dirname(__DIR__) . '/bs-data/config.php')) require dirname(__DIR__) . '/bs-data/config.php';
if (empty($TOKEN)) { http_response_code(500); exit('config.php mancante'); }

define('STORE', __DIR__ . '/dati_' . substr(hash('sha256', 'hellrise|' . $TOKEN), 0, 16));
if (!is_dir(STORE)) @mkdir(STORE, 0755, true);
if (!is_file(STORE . '/index.php')) @file_put_contents(STORE . '/index.php', "<?php http_response_code(404);\n");

if (!function_exists('array_is_list')) { function array_is_list(array $a) { $i = 0; foreach ($a as $k => $_) { if ($k !== $i++) return false; } return true; } }

const UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36';
const CLASSES = ['barbarian', 'druid', 'necromancer', 'rogue', 'sorcerer', 'spiritborn', 'paladin', 'warlock'];
const ROB_SHEET = '1bN8C_Fe9nrWv__jUc_Vdbl220g4NM0mlux-ga_s_-5s';

function check_token() {
    global $TOKEN;
    if (!isset($_GET['k']) || !hash_equals($TOKEN, (string)$_GET['k'])) { http_response_code(403); exit('accesso negato'); }
}

function load_json($file, $default) {
    $p = STORE . '/' . $file;
    if (!is_file($p)) return $default;
    $d = json_decode((string)file_get_contents($p), true);
    return is_array($d) ? $d : $default;
}

function save_json($file, $data) {
    $p = STORE . '/' . $file;
    file_put_contents($p . '.tmp', json_encode($data, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
    rename($p . '.tmp', $p);
}

function fetch($url, $timeout = 15) {
    $ch = curl_init($url);
    curl_setopt_array($ch, [CURLOPT_RETURNTRANSFER => true, CURLOPT_TIMEOUT => $timeout, CURLOPT_ENCODING => '',
        CURLOPT_FOLLOWLOCATION => true, CURLOPT_MAXREDIRS => 4,
        CURLOPT_HTTPHEADER => ['User-Agent: ' . UA, 'Accept: text/html,application/json;q=0.9,*/*;q=0.8', 'Accept-Language: en-US,en;q=0.8']]);
    $b = curl_exec($ch);
    $code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $GLOBALS['LAST_HTTP'] = $code == 200 ? '' : ('HTTP ' . $code . ($code ? '' : ' ' . curl_error($ch)));
    curl_close($ch);
    return $code == 200 && is_string($b) ? $b : null;
}

function note_error($src, $msg) {
    $st = &$GLOBALS['STATE'];
    $st['errors'][$src] = date('Y-m-d H:i') . ' ' . $msg;
}
function clear_error($src) { unset($GLOBALS['STATE']['errors'][$src]); }

// ---------- stagioni: file s15.json, s16.json… (si tengono solo le due più recenti) ----------
function season_file($n) { return 's' . (int)$n . '.json'; }

function season_put($n, $key, $value) {
    $n = (int)$n;
    if ($n < 1 || $n > 99) return;
    $d = load_json(season_file($n), ['season' => $n]);
    $d[$key] = $value;
    $d['updated'][$key] = time();
    save_json(season_file($n), $d);
    prune_seasons();
}

function seasons_list() {
    $out = [];
    foreach (glob(STORE . '/s*.json') ?: [] as $f) if (preg_match('/\/s(\d+)\.json$/', $f, $m)) $out[] = (int)$m[1];
    rsort($out);
    return $out;
}

function prune_seasons() {
    $all = seasons_list();
    foreach (array_slice($all, 2) as $old) @unlink(STORE . '/' . season_file($old));
}

function live_season() { return (int)($GLOBALS['STATE']['live'] ?? 0); }

function class_of($text) {
    $t = strtolower($text);
    $alias = ['barb' => 'barbarian', 'necro' => 'necromancer', 'sorc' => 'sorcerer', 'pala' => 'paladin', 'pal ' => 'paladin', 'hammerdin' => 'paladin', 'auradin' => 'paladin'];
    foreach (CLASSES as $c) if (strpos($t, $c) !== false) return $c;
    foreach ($alias as $a => $c) if (strpos($t . ' ', $a) !== false) return $c;
    return '';
}

function clean($s) { return trim(preg_replace('/\s+/u', ' ', html_entity_decode(strip_tags((string)$s), ENT_QUOTES | ENT_HTML5, 'UTF-8'))); }

// ======================================================================
// Orari dei boss del mondo (helltides.com: dati già pronti nella pagina, formato Nuxt "devalue")
function nuxt_decode($json) {
    $arr = json_decode($json, true);
    if (!is_array($arr)) return null;
    $memo = [];
    $res = function ($i, $depth = 0) use (&$res, &$arr, &$memo) {
        if (!is_int($i)) return $i;
        if ($i < 0 || !array_key_exists($i, $arr) || $depth > 60) return null;
        if (array_key_exists($i, $memo)) return $memo[$i];
        $v = $arr[$i];
        if (is_array($v)) {
            if (array_is_list($v)) {
                if (isset($v[0]) && is_string($v[0]) && in_array($v[0], ['ShallowReactive', 'Reactive', 'Ref', 'ShallowRef'], true)) { $out = $res($v[1], $depth + 1); }
                elseif (isset($v[0]) && is_string($v[0]) && in_array($v[0], ['Date', 'Set', 'Map', 'null', 'NaN'], true)) { $out = $v[0] === 'Date' ? ($v[1] ?? null) : null; }
                else { $out = []; foreach ($v as $x) $out[] = $res($x, $depth + 1); }
            } else { $out = []; foreach ($v as $k => $x) $out[$k] = $res($x, $depth + 1); }
        } else $out = $v;
        $memo[$i] = $out;
        return $out;
    };
    return $res(0);
}

function step_events() {
    $html = fetch('https://helltides.com/worldboss');
    if (!$html) { note_error('eventi', 'helltides.com non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')'); return false; }
    if (!preg_match('/<script[^>]*id="__NUXT_DATA__"[^>]*>(.*?)<\/script>/s', $html, $m)) { note_error('eventi', 'formato pagina cambiato'); return false; }
    $root = nuxt_decode($m[1]);
    $list = [];
    $walk = function ($o) use (&$walk, &$list) {
        if (!is_array($o)) return;
        if (isset($o['type'], $o['boss'], $o['startTime']) && $o['type'] === 'world_boss') {
            $zones = [];
            foreach (($o['zone'] ?? []) as $z) if (is_array($z) && !empty($z['name'])) $zones[] = ['id' => $z['id'] ?? '', 'name' => $z['name'], 'boss' => is_string($z['boss'] ?? null) ? $z['boss'] : ''];
            $list[] = ['t' => strtotime($o['startTime']), 'boss' => $o['boss'], 'zones' => $zones];
            return;
        }
        foreach ($o as $v) $walk($v);
    };
    $walk($root);
    $seen = [];
    $list = array_values(array_filter($list, function ($e) use (&$seen) { if (!$e['t'] || isset($seen[$e['t']])) return false; $seen[$e['t']] = 1; return true; }));
    usort($list, function ($a, $b) { return $a['t'] <=> $b['t']; });
    if (!$list) { note_error('eventi', 'nessun boss trovato nella pagina'); return false; }
    save_json('events.json', ['at' => time(), 'source' => 'helltides.com', 'worldBoss' => $list]);
    clear_error('eventi');
    return true;
}

// ======================================================================
// Maxroll: elenco delle guide (stagione, tipo, autore) + id del planner di ogni guida
function step_maxroll_list() {
    $html = fetch('https://maxroll.gg/d4/build-guides');
    if (!$html) { note_error('maxroll', 'elenco guide non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')'); return false; }
    $parts = preg_split('/<div role="row"/', $html);
    $rows = [];
    $maxSeason = 0;
    foreach ($parts as $p) {
        if (!preg_match('/<a href="\/d4\/build-guides\/([a-z0-9\-]+)"[^>]*aria-label="([^"]*)"/', $p, $m)) continue;
        $slug = $m[1];
        $title = clean($m[2]);
        preg_match_all('/_CategoryTag__label[^"]*">([^<]+)</', $p, $tags);
        $tags = array_map('clean', $tags[1] ?? []);
        $season = 0; $type = '';
        foreach ($tags as $t) {
            if (preg_match('/Season (\d+)/i', $t, $sm)) $season = (int)$sm[1];
            elseif (preg_match('/leveling|endgame|starter|speedfarm|push/i', $t)) $type = $t;
        }
        if (!$type) $type = stripos($title, 'leveling') !== false ? 'Leveling' : 'Endgame';
        $author = preg_match('/_author[^"]*">([^<]+)</', $p, $am) ? clean($am[1]) : '';
        $upd = preg_match('/datePrefix[^>]*>[^<]*<\/span>([^<]+)</', $p, $dm) ? strtotime(clean($dm[1])) : 0;
        $rows[$slug] = ['slug' => $slug, 'title' => $title, 'cls' => class_of($slug . ' ' . $title), 'season' => $season, 'type' => $type,
            'author' => $author, 'updated' => $upd ?: 0, 'url' => 'https://maxroll.gg/d4/build-guides/' . $slug];
        $maxSeason = max($maxSeason, $season);
    }
    if (!$rows) { note_error('maxroll', 'nessuna guida trovata (pagina cambiata?)'); return false; }
    if ($maxSeason) $GLOBALS['STATE']['live'] = max(live_season(), $maxSeason);
    // conserva gli id del planner già trovati
    $old = load_json('maxroll_guides.json', []);
    foreach ($rows as $slug => &$r) {
        if (isset($old[$slug]['profile']) && ($old[$slug]['updated'] ?? 0) == $r['updated']) $r['profile'] = $old[$slug]['profile'];
    }
    unset($r);
    save_json('maxroll_guides.json', $rows);
    clear_error('maxroll');
    publish_maxroll();
    return true;
}

// pagine delle guide ancora senza id del planner (alcune per esecuzione)
function step_maxroll_pages($max, $deadline) {
    $rows = load_json('maxroll_guides.json', []);
    $done = 0; $changed = false;
    foreach ($rows as $slug => &$r) {
        if (array_key_exists('profile', $r)) continue;
        if ($done >= $max || microtime(true) > $deadline) break;
        $done++;
        $html = fetch($r['url'], 20);
        if (!$html) { $r['profile'] = null; $changed = true; continue; }
        $r['profile'] = preg_match('/data-d4-profile="([a-z0-9]+)"/i', $html, $m) ? $m[1] : null;
        if (preg_match('/<title>([^<]+)<\/title>/', $html, $tm)) $r['pageTitle'] = clean($tm[1]);
        if (preg_match('/<meta[^>]+name="description"[^>]+content="([^"]*)"/', $html, $dm)) $r['desc'] = mb_substr(clean($dm[1]), 0, 300);
        if (preg_match('/levelingRelatedPosts[^\]]*?(\/d4\/build-guides\/[a-z0-9\-]+)/', $html, $lm)) $r['leveling'] = basename($lm[1]);
        $changed = true;
        unset($html);
    }
    unset($r);
    if ($changed) { save_json('maxroll_guides.json', $rows); publish_maxroll(); }
    return $done;
}

function publish_maxroll() {
    $rows = load_json('maxroll_guides.json', []);
    $by = [];
    foreach ($rows as $r) if ($r['season']) $by[$r['season']][] = $r;
    foreach ($by as $n => $list) {
        if ($n < live_season() - 1) continue;
        usort($list, function ($a, $b) { return $b['updated'] <=> $a['updated']; });
        season_put($n, 'maxroll', $list);
    }
}

// ======================================================================
// Tier list di Rob2628 (Google Sheets pubblico, letto come CSV)
function rob_csv($sheet) {
    $url = 'https://docs.google.com/spreadsheets/d/' . ROB_SHEET . '/gviz/tq?tqx=out:csv&headers=0&sheet=' . rawurlencode($sheet);
    $csv = fetch($url);
    if (!$csv || stripos($csv, '<html') !== false) return null;
    $lines = [];
    $fh = fopen('php://memory', 'r+');
    fwrite($fh, $csv);
    rewind($fh);
    while (($row = fgetcsv($fh, 0, ',', '"', '')) !== false) $lines[] = $row;
    fclose($fh);
    return $lines;
}

// righe: [col A note, col B tier, col C build, col D info, col E planner, …]
function rob_parse($lines, $leveling) {
    $out = []; $tier = ''; $started = false; $tierLabel = '';
    foreach ($lines as $i => $r) {
        $b = trim($r[1] ?? ''); $c = trim($r[2] ?? ''); $d = trim($r[3] ?? ''); $e = trim($r[4] ?? '');
        if ($i === 0) continue;
        if (preg_match('/^(S\+|S|A|B|C|D|F)$/', $b)) { $tier = $b; $tierLabel = ''; }
        elseif ($b !== '' && $tier !== '' && !$tierLabel) $tierLabel = $b; // es. "Speed 150", "Fastest to 70"
        if ($c === '' || $c === 'Build' || stripos($c, 'many more') !== false || stripos($c, 'more crazy') !== false) continue;
        $url = preg_match('/^https?:\/\//', $e) ? $e : '';
        $out[] = ['tier' => $tier ?: '?', 'tierNote' => $tierLabel, 'build' => $c, 'info' => $d, 'url' => $url, 'cls' => class_of($c . ' ' . $url), 'rank' => count($out) + 1];
        $started = true;
    }
    return $out;
}

// nomi dei fogli nell'ordine del file (il primo è quello che Google restituisce se il nome non esiste)
function rob_sheet_names() {
    $html = fetch('https://docs.google.com/spreadsheets/d/' . ROB_SHEET . '/htmlview', 20);
    if (!$html) return [];
    preg_match_all('/items\.push\(\{name: "([^"]+)"/', $html, $m);
    $out = [];
    foreach ($m[1] as $n) { $n = clean($n); if (!in_array($n, $out, true)) $out[] = $n; }
    return $out;
}

function step_rob() {
    $live = live_season();
    if (!$live) return false;
    $names = rob_sheet_names();
    $default = rob_csv('zz-foglio-inesistente');
    $defKey = $default ? md5(json_encode($default)) : '';
    $exists = function ($name, $lines) use ($names, $defKey) {
        if (!$lines || count($lines) < 3) return false;
        if ($names) return in_array($name, $names, true);
        return md5(json_encode($lines)) !== $defKey; // senza elenco: diverso dal foglio predefinito
    };
    $any = false;
    foreach ([$live + 1, $live, $live - 1] as $n) {
        if ($n < 1 || ($n === $live - 1 && $any && is_file(STORE . '/' . season_file($live + 1)))) continue;
        $res = ['sheet' => '', 'tier' => [], 'leveling' => [], 'levelingSheet' => '', 'source' => 'https://docs.google.com/spreadsheets/d/' . ROB_SHEET . '/htmlview'];
        foreach (["S$n Tier List", "S$n Tier List WIP", "S$n Build Tier List", "S$n Best Builds"] as $name) {
            if ($names && !in_array($name, $names, true)) continue;
            $lines = rob_csv($name);
            if ($exists($name, $lines)) { $res['sheet'] = $name; $res['tier'] = rob_parse($lines, false); break; }
        }
        foreach (["S$n Leveling Tier List", "S$n Leveling Tier List WIP"] as $name) {
            if ($names && !in_array($name, $names, true)) continue;
            $lines = rob_csv($name);
            if ($exists($name, $lines)) { $res['levelingSheet'] = $name; $res['leveling'] = rob_parse($lines, true); break; }
        }
        if ($res['tier'] || $res['leveling']) {
            $res['wip'] = stripos($res['sheet'] . $res['levelingSheet'], 'WIP') !== false;
            season_put($n, 'rob', $res);
            $any = true;
        }
    }
    $any ? clear_error('rob') : note_error('rob', 'tier list non trovata (' . $GLOBALS['LAST_HTTP'] . ')');
    return $any;
}

// ======================================================================
// D4Builds: elenco build (pagina iniziale) + tabelle loot (live e PTR)
function step_d4builds() {
    $live = live_season();
    $html = fetch('https://d4builds.gg/');
    $builds = [];
    if ($html) {
        preg_match_all('/href="\/builds\/([a-z0-9\-]+)\/"/', $html, $m);
        foreach (array_unique($m[1]) as $slug) {
            if (preg_match('/^[0-9a-f]{8}-/', $slug)) continue; // build senza nome leggibile
            $type = preg_match('/-leveling$/', $slug) ? 'Leveling' : 'Endgame';
            $name = ucwords(str_replace('-', ' ', preg_replace('/-(endgame|leveling)$/', '', $slug)));
            $builds[] = ['slug' => $slug, 'name' => $name, 'cls' => class_of($slug), 'type' => $type, 'url' => 'https://d4builds.gg/builds/' . $slug . '/'];
        }
    }
    if ($builds && $live) season_put($live, 'd4builds', $builds);
    // tabelle loot: sono nei dati di ogni pagina build
    $pd = json_decode((string)fetch('https://d4builds.gg/page-data/builds/whirlwind-barbarian-endgame/page-data.json', 20), true);
    $pc = $pd['result']['pageContext'] ?? null;
    if (!$pc) { note_error('d4builds', 'tabelle loot non raggiungibili (' . $GLOBALS['LAST_HTTP'] . ')'); return (bool)$builds; }
    $norm = function ($tables) {
        $out = [];
        foreach (($tables ?: []) as $t) {
            $by = [];
            foreach (CLASSES as $c) if (!empty($t[$c])) $by[$c] = array_values($t[$c]);
            $out[] = ['boss' => $t['boss'] ?? '', 'name' => $t['name'] ?? ($t['boss'] ?? ''), 'location' => $t['location'] ?? '', 'all' => array_values($t['all'] ?? []), 'byClass' => (object)$by];
        }
        return $out;
    };
    $liveT = $norm($pc['lootTables'] ?? []);
    $ptrT = $norm($pc['lootTablesPTR'] ?? []);
    if ($live && $liveT) season_put($live, 'loot', ['source' => 'd4builds.gg', 'bosses' => $liveT]);
    if ($live && $ptrT && json_encode($ptrT) !== json_encode($liveT)) season_put($live + 1, 'loot', ['source' => 'd4builds.gg (PTR)', 'ptr' => true, 'bosses' => $ptrT]);
    clear_error('d4builds');
    return true;
}

// ======================================================================
// Maxroll: tabella "chi droppa cosa" (Boss Loot Table Cheat Sheet), con gli id degli oggetti
function step_maxroll_loot() {
    $live = live_season();
    $html = fetch('https://maxroll.gg/d4/resources/boss-loot-table-cheat-sheet', 25);
    if (!$html || !$live) { note_error('maxroll-loot', 'pagina non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')'); return false; }
    libxml_use_internal_errors(true);
    $doc = new DOMDocument();
    $doc->loadHTML('<?xml encoding="utf-8"?>' . $html, LIBXML_NOWARNING | LIBXML_NOERROR | LIBXML_COMPACT);
    unset($html);
    $xp = new DOMXPath($doc);
    $sections = []; $cur = null; $stop = false;
    foreach ($xp->query('//h2 | //h3 | //*[@data-d4-id]') as $el) {
        if ($stop) break;
        $tag = strtolower($el->nodeName);
        if ($tag === 'h2') {
            $name = clean($el->textContent);
            if (preg_match('/^(summary|changelog|credits)/i', $name)) { $stop = true; continue; }
            $sections[] = ['name' => $name, 'items' => []];
            $cur = count($sections) - 1;
            continue;
        }
        if ($tag === 'h3') {
            $t = clean($el->textContent);
            if ($cur !== null && preg_match('/mythic/i', $t)) { $sections[] = ['name' => $t, 'items' => []]; $cur = count($sections) - 1; }
            continue;
        }
        if ($cur === null) continue;
        $id = (int)$el->getAttribute('data-d4-id');
        $name = $el->getAttribute('data-d4-text') ?: clean($el->textContent);
        if (!$id || $name === '') continue;
        $sections[$cur]['items'][$id] = ['id' => $id, 'name' => $name];
    }
    $out = [];
    foreach ($sections as $s) if ($s['items']) $out[] = ['name' => $s['name'], 'items' => array_values($s['items'])];
    if (!$out) { note_error('maxroll-loot', 'nessun boss trovato (pagina cambiata?)'); return false; }
    season_put($live, 'lootMaxroll', ['source' => 'maxroll.gg', 'sections' => $out]);
    clear_error('maxroll-loot');
    return true;
}

// ======================================================================
// Mobalytics: build in evidenza per classe (dati già presenti nella pagina)
function moba_docs($html) {
    $i = strpos($html, 'window.__PRELOADED_STATE__=');
    if ($i === false) return [];
    $i += strlen('window.__PRELOADED_STATE__=');
    $end = strpos($html, '</script>', $i);
    $json = rtrim(trim(substr($html, $i, $end - $i)), ';');
    $st = json_decode($json, true);
    unset($json);
    if (!is_array($st)) return [];
    $docs = [];
    $walk = function ($o, $d) use (&$walk, &$docs) {
        if ($d > 30 || !is_array($o)) return;
        if (isset($o['slugifiedName'], $o['tags'], $o['data']) && is_array($o['data'])) { $docs[] = $o; return; }
        foreach ($o as $v) if (is_array($v)) $walk($v, $d + 1);
    };
    $walk($st, 0);
    $out = [];
    foreach ($docs as $doc) {
        $tags = [];
        foreach (($doc['tags']['data'] ?? []) as $t) $tags[$t['groupSlug'] ?? ''][] = $t['slug'] ?? '';
        if (empty($tags['class'])) continue; // guide e notizie
        $season = 0;
        foreach (($tags['season'] ?? []) as $s) if (preg_match('/(\d+)/', $s, $m)) $season = max($season, (int)$m[1]);
        $slug = $doc['featured']['slug'] ?? '';
        $url = $slug ? 'https://mobalytics.gg/diablo-4/builds/' . $slug
            : 'https://mobalytics.gg/diablo-4/profile/' . ($doc['author']['shortName'] ?? ($doc['author']['user']['username'] ?? '')) . '/builds/' . $doc['slugifiedName'];
        $out[$doc['id'] ?? $url] = ['id' => $doc['id'] ?? '', 'name' => clean($doc['data']['name'] ?? ''), 'cls' => $tags['class'][0], 'season' => $season,
            'type' => in_array('leveling', $tags['build-type'] ?? [], true) ? 'Leveling' : 'Endgame',
            'activity' => $tags['activity'] ?? [], 'author' => $doc['author']['name'] ?? '', 'url' => $url,
            'verified' => in_array('verified', $tags['verified'] ?? [], true), 'likes' => (int)($doc['favorites']['favoritesCount'] ?? 0),
            'updated' => strtotime($doc['updatedAt'] ?? '') ?: 0];
    }
    return $out;
}

function step_mobalytics(&$es) {
    $pages = array_merge(['builds'], array_map(function ($c) { return $c . '-builds'; }, CLASSES));
    $i = (int)($es['i'] ?? 0);
    if ($i >= count($pages)) $i = 0;
    $html = fetch('https://mobalytics.gg/diablo-4/' . $pages[$i], 20);
    $acc = load_json('moba_acc.json', []);
    if ($html) {
        foreach (moba_docs($html) as $k => $d) $acc[$k] = $d;
        clear_error('mobalytics');
    } else note_error('mobalytics', $pages[$i] . ' non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')');
    unset($html);
    // tiene solo le ultime due stagioni
    $live = live_season();
    $acc = array_filter($acc, function ($d) use ($live) { return !$live || !$d['season'] || $d['season'] >= $live - 1; });
    save_json('moba_acc.json', $acc);
    $es['i'] = $i + 1;
    if ($es['i'] >= count($pages)) {
        $by = [];
        foreach ($acc as $d) $by[$d['season'] ?: $live][] = $d;
        foreach ($by as $n => $list) {
            usort($list, function ($a, $b) { return $b['likes'] <=> $a['likes']; });
            season_put($n, 'mobalytics', array_values($list));
        }
        return true; // giro completo
    }
    return false;
}

// ======================================================================
// Mobalytics: guida "Boss Loot Table" (dove si evoca ogni boss, materiali, tipo di danno)
function step_moba_bossinfo() {
    $live = live_season();
    $html = fetch('https://mobalytics.gg/diablo-4/guides/diablo-4-boss-loot-tables', 25);
    if (!$html || !$live) { note_error('mobalytics-boss', 'guida non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')'); return false; }
    libxml_use_internal_errors(true);
    $doc = new DOMDocument();
    $doc->loadHTML('<?xml encoding="utf-8"?>' . $html, LIBXML_NOWARNING | LIBXML_NOERROR | LIBXML_COMPACT);
    unset($html);
    $xp = new DOMXPath($doc);
    $out = []; $cur = null; $done = [];
    foreach ($xp->query('//h2 | //p | //li') as $el) {
        if (strtolower($el->nodeName) === 'h2') {
            $name = clean($el->textContent);
            if (preg_match('/table of contents|see more|introduction/i', $name)) { $cur = null; continue; }
            $cur = $name; $out[$cur] = []; $done[$cur] = false;
            continue;
        }
        if ($cur === null || $done[$cur]) continue;
        $t = clean($el->textContent);
        if ($t === '') continue;
        if (preg_match('/^(non-class specific|barbarian|weapons|equipment|jewell?ery)$/i', $t)) { $done[$cur] = true; continue; }
        if (count($out[$cur]) < 8 && mb_strlen($t) < 600) $out[$cur][] = $t;
    }
    $list = [];
    foreach ($out as $name => $lines) if ($lines) $list[] = ['name' => $name, 'lines' => $lines];
    if (!$list) { note_error('mobalytics-boss', 'nessun boss trovato (pagina cambiata?)'); return false; }
    season_put($live, 'bossInfo', ['source' => 'mobalytics.gg', 'url' => 'https://mobalytics.gg/diablo-4/guides/diablo-4-boss-loot-tables', 'bosses' => $list]);
    clear_error('mobalytics-boss');
    return true;
}

// ======================================================================
// Patch: note ufficiali (news.blizzard.com) + messaggi dello staff sul forum (hotfix, problemi noti)
function patch_sanitize($html) {
    $html = preg_replace('/<(script|style|iframe)[^>]*>.*?<\/\1>/is', '', $html);
    $html = strip_tags($html, '<h2><h3><h4><h5><p><ul><ol><li><strong><b><em><i><br><hr>');
    $html = preg_replace('/<(\w+)\s[^>]*>/', '<$1>', $html); // niente attributi
    return trim(preg_replace("/\n\s*\n+/", "\n", $html));
}

function step_patches() {
    $news = fetch('https://news.blizzard.com/en-us/diablo4', 25);
    if (!$news) { note_error('patch', 'news.blizzard.com non raggiungibile (' . $GLOBALS['LAST_HTTP'] . ')'); return false; }
    preg_match_all('/\/article\/(\d+)\/([a-z0-9\-]+)/', $news, $m, PREG_SET_ORDER);
    unset($news);
    $arts = [];
    foreach ($m as $x) {
        if (isset($arts[$x[1]])) continue;
        if (!preg_match('/patch-notes|ptr|hotfix/', $x[2])) continue;
        $arts[$x[1]] = $x[2];
    }
    $arts = array_slice($arts, 0, 4, true); // le più recenti (es. 3.3, PTR 3.3, 3.2)
    $articles = [];
    foreach ($arts as $id => $slug) {
        $html = fetch("https://news.blizzard.com/en-us/article/$id/$slug", 25);
        if (!$html) continue;
        $title = preg_match('/<title>([^<]+)<\/title>/', $html, $tm) ? clean(preg_replace('/\s+—.*$/u', '', $tm[1])) : $slug;
        $title = preg_replace('/\s*[—-]\s*(Diablo IV|Blizzard News).*$/u', '', $title);
        $patches = [];
        $parts = preg_split('/<div class="panel-title">/', $html);
        array_shift($parts);
        foreach ($parts as $p) {
            if (!preg_match('/<a[^>]*>(.*?)<\/a>/s', $p, $hm)) continue;
            $head = clean($hm[1]);
            $body = '';
            if (preg_match('/<div class="panel-body">(.*)/s', $p, $bm)) {
                $b = $bm[1];
                $cut = strpos($b, '<div class="panel">');
                if ($cut !== false) $b = substr($b, 0, $cut);
                $body = patch_sanitize($b);
            }
            $date = preg_match('/—\s*(.+)$/u', $head, $dm) ? strtotime($dm[1]) : 0;
            $ver = preg_match('/^([\d.]+[a-z]?)/', $head, $vm) ? $vm[1] : '';
            $patches[] = ['title' => $head, 'version' => $ver, 'date' => $date ?: 0, 'html' => $body];
        }
        if (!$patches && preg_match('/<section class="blog">(.*?)<\/section>/s', $html, $sm)) {
            $patches[] = ['title' => $title, 'version' => '', 'date' => 0, 'html' => mb_substr(patch_sanitize($sm[1]), 0, 200000)];
        }
        $articles[] = ['id' => $id, 'title' => $title, 'url' => "https://news.blizzard.com/en-us/article/$id/$slug", 'ptr' => strpos($slug, 'ptr') !== false, 'patches' => array_slice($patches, 0, 25)];
        unset($html);
    }
    // forum: messaggi dello staff Blizzard
    $blue = [];
    $j = json_decode((string)fetch('https://us.forums.blizzard.com/en/d4/groups/blizzard-tracker/posts.json', 20), true);
    foreach (($j['posts'] ?? []) as $p) {
        $blue[] = ['id' => $p['id'] ?? 0, 'title' => $p['topic_title'] ?? '', 'excerpt' => clean($p['excerpt'] ?? ''), 'at' => strtotime($p['created_at'] ?? '') ?: 0,
            'author' => $p['username'] ?? ($p['name'] ?? ''), 'url' => 'https://us.forums.blizzard.com/en/d4' . ($p['url'] ?? ''), 'topic' => $p['topic_id'] ?? 0];
    }
    // testo completo dei messaggi su hotfix / problemi noti
    $n = 0;
    foreach ($blue as &$b) {
        if ($n >= 6 || !preg_match('/hotfix|known issues|patch|update|maintenance|server/i', $b['title'])) continue;
        $pj = json_decode((string)fetch('https://us.forums.blizzard.com/en/d4/posts/' . (int)$b['id'] . '.json', 15), true);
        if (!empty($pj['cooked'])) $b['html'] = mb_substr(patch_sanitize($pj['cooked']), 0, 60000);
        $n++;
    }
    unset($b);
    if (!$articles && !$blue) { note_error('patch', 'nessuna patch trovata'); return false; }
    save_json('patches.json', ['at' => time(), 'articles' => $articles, 'forum' => $blue]);
    clear_error('patch');
    return true;
}
