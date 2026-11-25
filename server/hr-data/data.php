<?php
// Dati per Hellrise: ?k=TOKEN&what=index | season&n=16 | events
require_once __DIR__ . '/lib.php';
// lettura aperta: chiunque abbia Hellrise legge i dati (scrivere resta solo con il codice, in app.php)
header('Cache-Control: no-store');
$GLOBALS['STATE'] = load_json('state.json', ['live' => 0, 'last' => [], 'errors' => []]);
$st = $GLOBALS['STATE'];
$what = $_GET['what'] ?? 'index';

function out_json($body) {
    header('Content-Type: application/json; charset=utf-8');
    if (strpos($_SERVER['HTTP_ACCEPT_ENCODING'] ?? '', 'gzip') !== false && function_exists('gzencode') && !ini_get('zlib.output_compression')) {
        header('Content-Encoding: gzip');
        echo gzencode($body, 6);
    } else echo $body;
}

if ($what === 'season') {
    $f = STORE . '/' . season_file((int)($_GET['n'] ?? 0));
    if (!is_file($f)) { http_response_code(404); exit('stagione non disponibile'); }
    out_json(file_get_contents($f));
} elseif ($what === 'fullidx') {
    out_json(is_file(STORE . '/full_index.json') ? file_get_contents(STORE . '/full_index.json') : '{}');
} elseif ($what === 'full') {
    $id = preg_replace('/[^0-9a-f]/', '', strtolower((string)($_GET['id'] ?? '')));
    $f = STORE . '/full/' . $id . '.json.gz';
    if (strlen($id) !== 16 || !is_file($f)) { http_response_code(404); exit('build non ancora raccolta'); }
    header('Content-Type: application/json; charset=utf-8');
    header('X-Updated: ' . filemtime($f));
    if (strpos($_SERVER['HTTP_ACCEPT_ENCODING'] ?? '', 'gzip') !== false) { header('Content-Encoding: gzip'); readfile($f); }
    else echo gzdecode(file_get_contents($f));
    exit;
} elseif ($what === 'patches') {
    out_json(is_file(STORE . '/patches.json') ? file_get_contents(STORE . '/patches.json') : '{}');
} elseif ($what === 'events') {
    out_json(is_file(STORE . '/events.json') ? file_get_contents(STORE . '/events.json') : '{}');
} else {
    $seasons = [];
    foreach (seasons_list() as $n) {
        $d = load_json(season_file($n), []);
        $seasons[] = ['n' => $n, 'updated' => $d['updated'] ?? [], 'ptr' => $n > (int)$st['live'], 'live' => $n === (int)$st['live']];
    }
    out_json(json_encode(['live' => (int)$st['live'], 'seasons' => $seasons, 'lastRun' => $st['lastRun'] ?? 0, 'runs' => $st['runs'] ?? 0,
        'errors' => (object)($st['errors'] ?? []), 'events' => load_json('events.json', (object)[]), 'patchesAt' => (int)(load_json('patches.json', [])['at'] ?? 0), 'php' => PHP_VERSION], JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES));
}

// niente cron? se i dati sono vecchi, lavora un po' dopo aver risposto all'app
if (time() - ($st['lastRun'] ?? 0) > 300) {
    if (function_exists('fastcgi_finish_request')) fastcgi_finish_request();
    else { @ob_end_flush(); @flush(); }
    define('HR_INTERNAL', 1);
    ob_start();
    include __DIR__ . '/run.php';
    ob_end_clean();
}
