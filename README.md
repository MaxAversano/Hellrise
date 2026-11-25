# Hellrise

Companion per Diablo IV (Windows, exe portable): build (Maxroll, D4Builds, Mobalytics) con la tier list di Rob2628,
"chi droppa cosa", timer degli eventi con mappa, patch e hotfix, profili macro suggeriti per D4Pressure e overlay in gioco
(timer e guida paragon). Dati divisi per stagione (la precedente e l'ultima), raccolti dal server `hr-data`.

Compilare (Go 1.24+):

```
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H windowsgui" -o Hellrise.exe .
```

Hellrise non è affiliato con Blizzard Entertainment.
