# FlightCLI

```text
███████╗██╗     ██╗ ██████╗ ██╗  ██╗████████╗ ██████╗██╗     ██╗
██╔════╝██║     ██║██╔════╝ ██║  ██║╚══██╔══╝██╔════╝██║     ██║
█████╗  ██║     ██║██║  ███╗███████║   ██║   ██║     ██║     ██║
██╔══╝  ██║     ██║██║   ██║██╔══██║   ██║   ██║     ██║     ██║
██║     ███████╗██║╚██████╔╝██║  ██║   ██║   ╚██████╗███████╗██║
╚═╝     ╚══════╝╚═╝ ╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝╚══════╝╚═╝
```

FlightCLI is a terminal flight tracker powered by the [AviationStack API](https://aviationstack.com/).

It supports two ways of working:

- interactive TUI mode with `flightcli`
- direct CLI commands like `flightcli status AA100`

## Features

- interactive terminal UI for flight lookup, airport boards, and route search
- live flight status snapshots
- airport departures and arrivals boards
- route search between two airports
- live refresh mode with `track`
- optional JSON output for snapshot commands
- local disk cache for repeat lookups

## Requirements

- Go 1.25+
- an [AviationStack](https://aviationstack.com/) API key

## Setup

1. Get an API key from [AviationStack](https://aviationstack.com/).

2. Set `AVIATIONSTACK_API_KEY`.

Create a `.env` file in the project root:

```env
AVIATIONSTACK_API_KEY=your_key_here
```

Or set it in your shell.

PowerShell:

```powershell
$env:AVIATIONSTACK_API_KEY="your_key_here"
```

Bash:

```bash
export AVIATIONSTACK_API_KEY=your_key_here
```

3. Build the app.

PowerShell:

```powershell
go build -o flightcli.exe .
```

Bash:

```bash
go build -o flightcli .
```

You can also run it without a build:

```bash
go run .
```

## Usage

### Interactive TUI

Launch the full-screen terminal UI:

PowerShell:

```powershell
.\flightcli.exe
```

Bash:

```bash
./flightcli
```

Inside the TUI you can:

- track a flight by number
- open an airport departures or arrivals board
- search a route between two airports
- refresh the current result

Use the on-screen hints for controls. `q` quits.

### CLI commands

#### Flight status

```bash
flightcli status AA100
flightcli status KE038 --json
```

This returns airline, route, status, timestamps, and live telemetry when available.

#### Airport board

Departures are the default:

```bash
flightcli airport JFK
flightcli airport JFK --type departures
flightcli airport JFK --type arrivals
```

#### Route search

```bash
flightcli search --from JFK --to LAX
flightcli search --from SIN --to NRT --json
```

#### Live tracking

```bash
flightcli track AA100 --interval 30
```

This continuously refreshes the selected flight until you stop it with `Ctrl+C`.

## Notes

- `flightcli` with no subcommand opens the interactive TUI.
- `--json` is supported for snapshot commands like `status`, `airport`, and `search`.
- `--json` is not supported with `track`.
- Airport inputs must be valid 3-letter IATA codes.
- Requests are sent over HTTPS.
- Cached responses may show a `(cached)` indicator.

## Built With

- [Go](https://go.dev/)
- [Cobra](https://github.com/spf13/cobra)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [AviationStack API](https://aviationstack.com/)

## License

[MIT](LICENSE)
