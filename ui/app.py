from flask import Flask, render_template
import requests

GO_BACKEND_URL = "http://localhost:8080"

app = Flask(__name__)

def fetch_json(endpoint):
    try:
        resp = requests.get(f"{GO_BACKEND_URL}{endpoint}", timeout=2)
        resp.raise_for_status()
        return resp.json()
    except Exception as e:
        return {"error": str(e)}

@app.route("/")
def dashboard():
    health = fetch_json("/health")
    metrics = fetch_json("/metrics")
    peers = fetch_json("/admin/peers")

    return render_template(
        "dashboard.html",
        health=health,
        metrics=metrics,
        peers=peers,
    )

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)
