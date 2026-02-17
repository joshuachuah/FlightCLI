# flightcli

A command-line tool to track live flights and view airport departures/arrivals, powered by the [AviationStack API](https://aviationstack.com/).

## Setup

1. **Get an API key** from [AviationStack](https://aviationstack.com/) (free tier available).

2. **Set the environment variable:**

   Create a `.env` file in the project root:
   ```
   AVIATIONSTACK_API_KEY=your_key_here
   ```

   Or export it directly:
   ```bash
   export AVIATIONSTACK_API_KEY=your_key_here
   ```

3. **Build:**
   ```bash
   go build -o flightcli .
   ```

## Usage

### Flight status

Track a live flight by its IATA flight number:

```bash
flightcli status AA100
```

Output includes airline, route, status, flight time, remaining time (if in-flight), and live position data when available.

### Airport departures/arrivals

View departures or arrivals for an airport:

```bash
# Departures (default)
flightcli airport JFK

# Arrivals
flightcli airport JFK --type arrivals
```

## Built With

- [Go](https://go.dev/)
- [Cobra](https://github.com/spf13/cobra)
- [AviationStack API](https://aviationstack.com/)

## License

[MIT](LICENSE)
