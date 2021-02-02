subinclude("///pleasings2//kubernetes")

helm_chart(
    name = "chart",
)

helm_package(
    name = "pipeline",
    chart = ":chart",
)

helm_lint(
    name = "lint",
    chart = ":chart",
)
