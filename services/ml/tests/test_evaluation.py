from app.evaluation import evaluate_model


def test_synthetic_validation_evaluation_is_bounded() -> None:
    result = evaluate_model()

    assert result.normal_count == 120
    assert result.anomaly_count == 30
    assert result.recall >= 0.90
    assert result.false_positive_rate <= 0.15
