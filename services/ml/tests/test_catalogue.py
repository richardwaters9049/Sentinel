from datetime import timezone

import pytest

from app.catalogue import PROFILES, Profile, catalogue, contrast_cases
from app.catalogue_evaluation import evaluate_catalogue
from app.features import FEATURE_NAMES, extract_features
from app.model import BehaviourModel


@pytest.fixture(scope="module")
def model() -> BehaviourModel:
    return BehaviourModel()


@pytest.mark.parametrize("profile", PROFILES, ids=lambda profile: profile.id)
def test_profile_cases_are_reproducible_and_explainable(profile: Profile, model: BehaviourModel) -> None:
    reference = contrast_cases(profile.id, False)
    changed = contrast_cases(profile.id, True)
    assert reference == contrast_cases(profile.id, False)
    assert changed == contrast_cases(profile.id, True)
    assert len(reference) == len(changed) == 12
    assert set(profile.features) <= set(FEATURE_NAMES)
    assert profile.required_telemetry and profile.limitations and profile.analyst_actions
    assert {row.event_id for row in reference}.isdisjoint(row.event_id for row in changed)
    for row in reference + changed:
        assert row.timestamp.tzinfo == timezone.utc
        context = row.baseline
        assert context.prior_events_24h >= context.prior_events_60m
        assert context.auth_failures_60m <= context.prior_events_60m
        assert context.ot_events_24h <= context.prior_events_24h
        assert context.destination_diversity_24h == pytest.approx(
            context.unique_destination_ips_24h / context.prior_events_24h
        )
        assert context.auth_failure_rate_60m == pytest.approx(
            context.auth_failures_60m / context.prior_events_60m
        )
    reference_scores = [model.score(row).anomaly_score for row in reference]
    changed_scores = [model.score(row) for row in changed]
    assert sum(result.anomaly_score for result in changed_scores) > sum(reference_scores)
    explained = {item.feature for result in changed_scores for item in result.explanations}
    assert set(profile.features) & explained
    # Positive/contrast cases preserve unrelated feature values, except volume ratios and OT context.
    if profile.id in {"BA-001", "BA-002", "BA-004"}:
        for left, right in zip(reference, changed):
            a, b = extract_features(left).as_list(), extract_features(right).as_list()
            for feature, original, altered in zip(FEATURE_NAMES, a, b):
                if feature not in profile.features:
                    assert original == altered


def test_catalogue_reports_weak_isolated_coverage(model: BehaviourModel) -> None:
    report = evaluate_catalogue(model=model)
    profiles = {entry["profile_id"]: entry for entry in report["profiles"]}
    assert set(profiles) == {profile.id for profile in PROFILES}
    for id in ("BA-001", "BA-003", "BA-004"):
        assert profiles[id]["changed_flagged"] == 0
    for id in ("BA-002", "BA-005"):
        assert profiles[id]["changed_flagged"] == 12
    assert all(entry["reference_flagged"] == 0 for entry in profiles.values())
    assert report == evaluate_catalogue(model=model)
    assert catalogue()["model_version"] == report["model_version"]


def test_threshold_policy_changes_flags_not_scores(model: BehaviourModel) -> None:
    low = evaluate_catalogue(1, model)
    high = evaluate_catalogue(99, model)
    for a, b in zip(low["profiles"], high["profiles"]):
        assert a["reference_mean_score"] == b["reference_mean_score"]
        assert a["changed_mean_score"] == b["changed_mean_score"]
        assert a["changed_flagged"] >= b["changed_flagged"]


@pytest.mark.parametrize("threshold", [0, 100, -1, True, 65.5])
def test_invalid_threshold_rejected(threshold) -> None:
    with pytest.raises(ValueError):
        evaluate_catalogue(threshold)


def test_unknown_profile_rejected() -> None:
    with pytest.raises(ValueError):
        contrast_cases("not-a-profile", True)


def test_ot_reference_is_not_a_security_verdict(model: BehaviourModel) -> None:
    row = contrast_cases("BA-005", True)[0]
    # A known simulated authorised workflow can have the same feature vector.
    authorised = row.model_copy(update={"action": "authorised_maintenance", "event_id": "authorised-ot"})
    assert model.score(row).anomaly_score == model.score(authorised).anomaly_score
