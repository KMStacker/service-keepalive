# service-keepalive

A lightweight Go worker that automatically keeps free-tier cloud services active. It runs on a schedule via GitHub Actions to prevent idle timeouts and sleep modes.

## What It Does

- **Render**: Sends an HTTP GET request to keep a web service warm and prevent cold starts.
- **Aiven**: Queries the Aiven API for database status and automatically sends a power-on request if the service is in `POWERED_OFF` state.

## Configuration

The application requires the following environment variables (defined in `.env` for local runs or as GitHub repository Secrets for automated runs):

| Variable | Description |
| :--- | :--- |
| `RENDER_SITE_URL` | URL of the Render web service to ping (optional). |
| `AIVEN_API_TOKEN` | Authentication token for the Aiven REST API. |
| `AIVEN_PROJECT_NAME` | Aiven project identifier. |
| `AIVEN_SERVICE_NAME` | Aiven database service name. |

## Running Locally

1. Copy `.env.example` to `.env` (or create `.env`) and fill in your credentials.
2. Run the application:

```bash
go run main.go
```

## Automation

The worker is scheduled via GitHub Actions (`.github/workflows/keepalive.yaml`) to run every two days (`0 12 */2 * *`). It can also be triggered manually using the `workflow_dispatch` event.
