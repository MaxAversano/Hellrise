<?php
// Pagina per scaricare Hellrise (da mandare agli amici): https://www.aversano.design/hr-data/
header('Cache-Control: no-store');
$v = @json_decode((string)@file_get_contents(__DIR__ . '/app/version.json'), true) ?: [];
$ver = htmlspecialchars($v['version'] ?? '');
$mb = isset($v['size']) ? number_format($v['size'] / 1048576, 1, ',', '') : '';
$data = isset($v['at']) ? date('d/m/Y', (int)$v['at']) : '';
?><!DOCTYPE html>
<html lang="it"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Hellrise · companion per Diablo IV</title>
<style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;font:15px/1.55 "Segoe UI",system-ui,sans-serif;color:#f3e8e2;background:radial-gradient(1200px 700px at 20% 0%,#5a1d0c 0%,transparent 60%),radial-gradient(900px 600px at 100% 100%,#3a0f1f 0%,transparent 60%),#120807}
.card{width:min(560px,92vw);padding:34px 34px 28px;border-radius:24px;background:rgba(255,255,255,.06);border:1px solid rgba(255,255,255,.12);backdrop-filter:blur(18px);box-shadow:0 30px 80px rgba(0,0,0,.5)}
h1{margin:0;font:600 34px/1.1 Georgia,serif;letter-spacing:.02em}.sub{color:#c9b3aa;margin:6px 0 24px}
.logo{width:54px;height:54px;border-radius:16px;display:grid;place-items:center;background:linear-gradient(160deg,#ff9a52,#d9381e);margin-bottom:16px;font-size:28px}
a.btn{display:flex;align-items:center;justify-content:center;gap:10px;padding:15px 20px;border-radius:14px;background:linear-gradient(180deg,#ff8a3d,#e0501e);color:#1b0803;font-weight:700;font-size:16px;text-decoration:none;box-shadow:0 10px 30px -8px #ff6a2b}
a.btn:hover{filter:brightness(1.08)}.meta{text-align:center;color:#a8928a;font-size:13px;margin:10px 0 22px}
ol{padding-left:20px;margin:0;color:#e2d2cb}li{margin:6px 0}b{color:#fff}.note{margin-top:18px;font-size:13px;color:#a8928a}
</style></head><body><div class="card">
<div class="logo">🔥</div><h1>Hellrise</h1><div class="sub">Companion per Diablo IV: build, tier list, chi droppa cosa, eventi, overlay in gioco e filtri loot.</div>
<?php if ($ver): ?>
<a class="btn" href="app.php?get=1">⬇ Scarica Hellrise.exe</a>
<div class="meta">versione <?= $ver ?> · <?= $mb ?> MB · pubblicata il <?= $data ?></div>
<?php else: ?><div class="meta">Nessuna versione pubblicata per ora.</div><?php endif; ?>
<ol>
<li>Crea una cartella, per esempio <b>Desktop\Hellrise</b>, e metti lì dentro <b>Hellrise.exe</b> (accanto creerà la sua cartella dei dati).</li>
<li>Aprilo con doppio clic. Se Windows mostra “PC protetto da Windows”, premi <b>Ulteriori informazioni → Esegui comunque</b> (l'app non è firmata).</li>
<li>Fatto: build e dati arrivano dal server già pronti e l'app <b>si aggiorna da sola</b>.</li>
</ol>
<div class="note">Serve Windows 10 o 11. Hellrise non tocca il gioco e non invia tasti.</div>
</div></body></html>
