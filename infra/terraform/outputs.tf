output "ecs_cluster_arn" {
  value = aws_ecs_cluster.worker.arn
}

output "ecr_repository_url" {
  value = aws_ecr_repository.worker.repository_url
}

output "github_deploy_role_arn" {
  value = aws_iam_role.github_deploy.arn
}

output "human_deployer_role_arn" {
  value = aws_iam_role.human_deployer.arn
}

output "operator_user_arn" {
  value = aws_iam_user.operator.arn
}

output "event_rule_arns" {
  value = {
    import   = aws_cloudwatch_event_rule.import.arn
    update   = aws_cloudwatch_event_rule.update.arn
    generate = aws_cloudwatch_event_rule.generate.arn
  }
}
