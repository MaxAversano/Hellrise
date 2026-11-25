<?php
// Diagnostica: apri questa pagina per controllare che la raccolta funzioni (nessun dato riservato)
header('Content-Type: text/plain; charset=utf-8');
echo "Hellrise · server dati\nPHP " . PHP_VERSION . " · curl: " . (function_exists('curl_init') ? 'si' : 'NO') . " · dom: " . (class_exists('DOMDocument') ? 'si' : 'NO') . "\n";
echo "scrittura cartella: " . (is_writable(__DIR__) ? 'si' : 'NO') . "\n";
require_once __DIR__ . '/lib.php';
$st = load_json('state.json', null);
if (!$st) { echo "nessuna esecuzione ancora: apri run.php?k=CODICE oppure avvia Hellrise\n"; exit; }
echo "stagione live: " . ($st['live'] ?? '?') . " · stagioni salvate: " . implode(', ', seasons_list()) . "\n";
echo "esecuzioni: " . ($st['runs'] ?? 0) . " · ultima: " . (!empty($st['lastRun']) ? date('Y-m-d H:i:s', $st['lastRun']) : 'mai') . "\n";
foreach (($st['last'] ?? []) as $k => $t) echo "  $k: " . date('Y-m-d H:i', $t) . "\n";
foreach (($st['errors'] ?? []) as $k => $e) echo "errore $k: $e\n";
if (is_file(STORE . '/last_error.txt')) echo "ultimo errore PHP: " . file_get_contents(STORE . '/last_error.txt') . "\n";
foreach (seasons_list() as $n) {
    $d = load_json(season_file($n), []);
    $parts = [];
    foreach (['maxroll', 'd4builds', 'mobalytics'] as $k) $parts[] = $k . ' ' . count($d[$k] ?? []);
    $parts[] = 'rob ' . count($d['rob']['tier'] ?? []) . '+' . count($d['rob']['leveling'] ?? []);
    $parts[] = 'loot ' . count($d['loot']['bosses'] ?? []) . '/' . count($d['lootMaxroll']['sections'] ?? []);
    echo "S$n: " . implode(' · ', $parts) . "\n";
}
