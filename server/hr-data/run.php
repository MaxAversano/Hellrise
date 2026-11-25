<?php
// Avviato da un cron (es. cron-job.org ogni 5 minuti) oppure da data.php quando i dati sono vecchi.
// Ogni esecuzione fa un po' di lavoro (massimo ~40 secondi), a turno sulle varie fonti.
require_once __DIR__ . '/lib.php';
if (!defined('HR_INTERNAL')) check_token();
@set_time_limit(90);
ignore_user_abort(true);
register_shutdown_function(function () {
    $e = error_get_last();
    if ($e && in_array($e['type'], [E_ERROR, E_PARSE, E_CORE_ERROR, E_COMPILE_ERROR], true)) {
        @file_put_contents(STORE . '/last_error.txt', date('c') . ' ' . $e['message'] . ' (' . basename($e['file']) . ':' . $e['line'] . ')');
    }
});
if (!headers_sent()) header('Content-Type: text/plain; charset=utf-8');

$lock = fopen(STORE . '/run.lock', 'c');
if (!flock($lock, LOCK_EX | LOCK_NB)) { echo "già in esecuzione\n"; return; }
@ini_set('memory_limit', '256M');

$deadline = microtime(true) + 40;
$GLOBALS['STATE'] = load_json('state.json', ['live' => 0, 'last' => [], 'errors' => [], 'runs' => 0, 'moba' => []]);
$st = &$GLOBALS['STATE'];
$st['runs'] = ($st['runs'] ?? 0) + 1;
$st['lastRun'] = time();
$now = time();
$due = function ($key, $every) use (&$st, $now) { return $now - ($st['last'][$key] ?? 0) >= $every; };
$log = [];

$jobs = [
    // chiave, ogni quanti secondi, funzione
    ['events', 600, function () { return step_events(); }],
    ['maxroll', 3 * 3600, function () { return step_maxroll_list(); }],
    ['rob', 3600, function () { return step_rob(); }],
    ['d4builds', 3 * 3600, function () { return step_d4builds(); }],
    ['maxrollLoot', 6 * 3600, function () { return step_maxroll_loot(); }],
    ['patches', 1800, function () { return step_patches(); }],
];
foreach ($jobs as [$key, $every, $fn]) {
    if (microtime(true) > $deadline) break;
    if (!$due($key, $every)) continue;
    if (!in_array($key, ['maxroll', 'events', 'patches'], true) && !$st['live']) continue; // prima serve sapere la stagione
    try { $ok = $fn(); } catch (Throwable $e) { $ok = false; note_error($key, $e->getMessage()); }
    // se fallisce riprova tra un quarto d'ora invece di aspettare il giro intero
    $st['last'][$key] = $ok ? $now : $now - $every + 900;
    $log[] = $key . ($ok ? ' ok' : ' errore');
    save_json('state.json', $st);
}
// pagine delle guide Maxroll (id del planner) e Mobalytics, un po' per volta
if ($st['live'] && microtime(true) < $deadline) {
    $n = step_maxroll_pages(6, $deadline);
    if ($n) $log[] = "maxroll pagine +$n";
}
if (false && $st['live'] && microtime(true) < $deadline && ($due('mobalytics', 3 * 3600) || ($st['moba']['i'] ?? 0) > 0)) { // Mobalytics risponde 403 al server: lo scarica l'app dal PC
    $es = $st['moba'] ?? [];
    $full = step_mobalytics($es);
    if ($full) { $es['i'] = 0; $st['last']['mobalytics'] = $now; }
    $st['moba'] = $es;
    $log[] = 'mobalytics ' . ($full ? 'completo' : 'pagina ' . $es['i']);
}
unset($st['errors']['mobalytics'], $st['errors']['mobalytics-boss']);
$st['seasons'] = seasons_list();
save_json('state.json', $st);
echo ($log ? implode("\n", $log) : 'niente da fare') . "\n";
