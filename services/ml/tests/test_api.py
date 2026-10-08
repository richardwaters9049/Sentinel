from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_health_reports_model_version() -> None:
    response = client.get("/health")

    assert response.status_code == 200
    payload = response.json()
    assert payload["status"] == "ok"
    assert payload["model_version"] == "sentinel-behaviour-iforest-v1"


def test_score_contract() -> None:
    response = client.post(
        "/v1/score",
        json={
            "event_id": "evt-api-1",
            "entity_id": "asset-api-1",
            "timestamp": "2026-01-10T02:15:00Z",
            "category": "authentication",
            "action": "login",
            "outcome": "failure",
            "source_zone": "corporate",
            "destination_zone": "ot",
            "destination_port": 502,
            "actor_type": "service_account",
        },
    )

    assert response.status_code == 200
    payload = response.json()
    assert payload["event_id"] == "evt-api-1"
    assert payload["model_kind"] == "IsolationForest"
    assert payload["threshold"] == 65
    assert isinstance(payload["explanations"], list)


def test_catalogue_includes_versioned_profiles_and_coverage() -> None:
    response = client.get("/v1/catalogue")
    assert response.status_code == 200
    payload = response.json()
    assert payload["catalogue_version"] == "sentinel-behaviour-catalogue-v1"
    assert len(payload["profiles"]) == 5
    assert payload["validation"]["threshold"] == 65
    assert payload["validation"]["model_version"] == payload["model_version"]
    assert {profile["id"] for profile in payload["profiles"]} == {
        entry["profile_id"] for entry in payload["validation"]["profiles"]
    }
