# Fahrradverleih

Lösung für **Aufgabe 1 – Teil c): Fahrradverleih** der Software-Engineering-Lehrveranstaltung: ein lauffähiger Prototyp eines Fahrradverleihs mit Web-Dashboard, änderbaren Tarifen und stundenweiser Preisberechnung.

## Funktionen

### Verleih

- **Dashboard:** verfügbare, ausgeliehene und gesamte Fahrräder sowie registrierte Personen, jeweils aus dem aktuellen Datenbestand
- **Fahrradverwaltung:** Fahrräder anlegen und bearbeiten, Verfügbarkeit einsehen
- **Personen:** Personen mit Name, Ausweistyp und Ausweisnummer anlegen und bearbeiten
- **Ausleihen:** Ausleihe mit Person, Fahrrad, Verleihbeginn und Verleihdauer anlegen, neue Personen dabei direkt im Formular registrieren, Fahrräder zurückgeben
- **Doppelausleihe verhindert:** Ein Fahrrad kann nicht gleichzeitig mehrfach aktiv ausgeliehen sein.

### Preise und Tarife

- **Tarife:** Tarife mit Stundenpreis über die Oberfläche anlegen und bearbeiten
- **Preisberechnung:** stundenweise Berechnung mit Anzeige des Preises bereits im Ausleihformular
- **Nachvollziehbarkeit:** Gespeicherte Ausleihen behalten ihren Preis auch nach einer Tarifänderung.

## Technologien

- **Backend:** Go, nur Standardbibliothek (`net/http`, `encoding/json`, `sync`, `embed`)
- **Frontend:** HTML, CSS und Vanilla JavaScript, ohne Build-Prozess
- **Datenhaltung:** thread-sicherer In-Memory-Store (`map` mit `sync.RWMutex`)
- **Kommunikation:** HTTP/JSON

## Projektstruktur

```text
fahrradverleih/
├── main.go              # Serverstart und Auslieferung des Frontends
├── go.mod
├── internal/
│   ├── domain/          # Datenmodelle, Geschäftsregeln, Preisberechnung
│   ├── store/           # Thread-sichere Datenhaltung und Demodaten
│   └── handler/         # HTTP-Routen und JSON
├── frontend/            # index.html, style.css, app.js
├── CLAUDE.md
└── README.md
```

Die Geschäftsregeln liegen in `domain` und kennen weder HTTP noch die Datenhaltung. `store` führt zusammengehörige Änderungen unter einer Sperre aus, `handler` übersetzt nur zwischen HTTP und `store`. Das Frontend greift ausschließlich über die HTTP-API auf Daten zu.

## Starten

Voraussetzung ist **Go 1.22 oder neuer**. Weitere Abhängigkeiten gibt es nicht.

```bash
go run .
```

Der Befehl wird im Projektordner ausgeführt. Danach ist die Anwendung unter <http://127.0.0.1:8080> oder <http://localhost:8080> erreichbar. Beenden lässt sie sich mit `Strg+C`.

Beim Start werden fiktive Demodaten geladen: 20 Fahrräder, 10 Personen, 3 Tarife und 3 aktive Ausleihen. Meldet der Start, dass die Adresse bereits verwendet wird, läuft der Server vermutlich schon in einem anderen Terminal.

## Tests

```bash
go vet ./...
go test ./...
go test -race ./...   # Prüft zusätzlich auf Data Races (benötigt unter Windows einen C-Compiler)
```

Die Tests decken die Geschäftsregeln, die Preisberechnung, die Datenhaltung einschließlich gleichzeitiger Zugriffe sowie die HTTP-Abläufe und die Dashboard-Zahlen nach Ausleihe und Rückgabe ab.

## Bedienung

### Fahrräder und Personen anlegen

1. **Fahrradverwaltung** oder **Personen** öffnen.
2. Oben rechts auf **Neues Fahrrad** oder **Neue Person** klicken, das Formular ausfüllen und bestätigen.
3. Zum Ändern in der Tabelle auf **Bearbeiten** klicken. Das Formular öffnet sich mit den vorhandenen Daten.

### Ausleihe anlegen

1. Im **Dashboard** oder unter **Ausleihen** auf **Neue Ausleihe** klicken.
2. Person, verfügbares Fahrrad und Tarif wählen. Ist die Person noch nicht registriert, im Formular auf **Neue Person** klicken und ihre Daten eingeben. Nach **Person anlegen** ist sie automatisch ausgewählt, und die bisherigen Angaben zur Ausleihe bleiben erhalten. Die Person bleibt auch dann registriert, wenn die Ausleihe danach nicht zustande kommt.
3. Verleihbeginn und Verleihdauer in ganzen Stunden eingeben. Der Gesamtpreis wird direkt angezeigt.
4. **Ausleihe anlegen** klicken. Das Fahrrad ist ab jetzt nicht mehr verfügbar, auch wenn der Verleihbeginn in der Zukunft liegt.

### Fahrrad zurückgeben

1. Unter **Ausleihen** in der Ansicht **Aktuell** bei der passenden Ausleihe auf **Zurückgeben** klicken und bestätigen.
2. Das Fahrrad ist sofort wieder verfügbar. Die Ausleihe erscheint in der Ansicht **Abgeschlossen**.

### Preise ändern

1. **Tarife** öffnen.
2. Bei einem Tarif auf **Bearbeiten** klicken oder mit **Neuer Tarif** einen Tarif anlegen.
3. Den Stundenpreis eingeben, zum Beispiel `6,50`, und speichern.

## Flexible Preise

Der Preis ergibt sich aus Verleihdauer und Stundenpreis des Tarifs:

```text
Gesamtpreis = Verleihdauer in Stunden × Stundenpreis
Beispiel:     3 Std. × 5,00 € = 15,00 €
```

- **Beispieltarife:** Kinder 3,00 €, Standard 5,00 € und Premium 8,00 € pro Stunde
- **Beträge in Cent:** Alle Beträge werden als ganze Cent gespeichert, um Rundungsfehler zu vermeiden.
- **Eine zentrale Berechnung:** Der Preis wird nur im Backend berechnet (`Tariff.Price`). Das Formular fragt ihn über `/api/price-quote` ab.
- **Preis bleibt erhalten:** Jede Ausleihe speichert beim Anlegen Tarifname, Stundenpreis und Gesamtpreis. Tarifänderungen gelten deshalb nur für neue Ausleihen.
- **Fester Preis:** Der Preis richtet sich nach der geplanten Dauer. Eine frühere oder spätere Rückgabe ändert ihn nicht.

## Einschränkungen

- **Datenverlust beim Neustart:** Alle Daten liegen nur im Arbeitsspeicher. Nach einem Neustart sind Änderungen verloren und die Demodaten werden neu geladen.
- **Kein Löschen:** Fahrräder, Personen und Tarife können nicht gelöscht werden.
- **Keine Reservierungen:** Ein Fahrrad ist vom Anlegen der Ausleihe bis zur Rückgabe gesperrt, unabhängig vom Verleihbeginn. Verleihbeginne in der Vergangenheit sind erlaubt, um Ausleihen nachtragen zu können.
- **Namensänderungen:** Geänderte Namen von Fahrrädern und Personen erscheinen auch bei früheren Ausleihen. Nur die Tarifdaten bleiben unverändert erhalten.
- **Keine Anmeldung:** Der Server ist nur lokal erreichbar, und die API weist Anfragen fremder Webseiten ab. Eine Benutzerverwaltung gibt es nicht. Der Prototyp ist nur für fiktive Daten gedacht und nicht für den Betrieb im Netzwerk.
- **Keine automatische Aktualisierung:** Änderungen aus einem anderen Browserfenster erscheinen erst beim nächsten Seitenwechsel oder nach der nächsten Aktion.
