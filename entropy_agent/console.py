from __future__ import annotations

# --- Optional nicer console output ---
try:
    from rich.console import Console
    from rich.panel import Panel

    console = Console()

    def print_panel(title: str, body: str) -> None:
        console.print(Panel(body, title=title, expand=False))

    def print_info(msg: str) -> None:
        console.print(f"[bold cyan]INFO[/bold cyan] {msg}")

    def print_warn(msg: str) -> None:
        console.print(f"[bold yellow]WARN[/bold yellow] {msg}")

    def print_err(msg: str) -> None:
        console.print(f"[bold red]ERROR[/bold red] {msg}")

except Exception:
    console = None

    def print_panel(title: str, body: str) -> None:
        print(f"\n== {title} ==\n{body}\n")

    def print_info(msg: str) -> None:
        print(f"[INFO] {msg}")

    def print_warn(msg: str) -> None:
        print(f"[WARN] {msg}")

    def print_err(msg: str) -> None:
        print(f"[ERROR] {msg}")
