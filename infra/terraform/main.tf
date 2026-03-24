locals {
  tags = {
    Project     = var.project
    Service     = var.service
    Environment = var.environment
    ManagedBy   = var.managed_by
    Owner       = var.owner
  }

  ecs_events_role_arn           = "arn:aws:iam::${var.aws_account_id}:role/${var.ecs_events_role_name}"
  ecs_task_execution_role_arn   = "arn:aws:iam::${var.aws_account_id}:role/${var.ecs_task_execution_role_name}"
  ecr_worker_repository_arn     = "arn:aws:ecr:${var.aws_region}:${var.aws_account_id}:repository/${var.ecr_repository_name}"
  github_deploy_role_name       = "vimcolorschemes-worker-github-actions-deploy-role"
  github_deploy_policy_ecr      = "vimcolorschemes-worker-ecr-push-policy"
  github_deploy_policy_schedule = "vimcolorschemes-worker-deploy-ecs-events-policy"
  human_deployer_role_name      = "vimcolorschemes-worker-human-deployer-role"
  operator_user_name            = "vimcolorschemes-worker-operator"
}

resource "aws_ecr_repository" "worker" {
  name                 = var.ecr_repository_name
  image_tag_mutability = "MUTABLE"
  force_delete         = false
  tags                 = merge(local.tags, { Purpose = "deployment" })
}

resource "aws_ecs_cluster" "worker" {
  name = var.ecs_cluster_name
  tags = local.tags
}

resource "aws_cloudwatch_log_group" "run_job" {
  name = var.cloudwatch_log_group_name
  tags = local.tags
}

resource "aws_secretsmanager_secret" "github_token" {
  name = "vimcolorschemes/worker/github_token"
  tags = merge(local.tags, { Purpose = "runtime-secret" })
}

resource "aws_secretsmanager_secret" "database_url" {
  name = "vimcolorschemes/worker/database_url"
  tags = merge(local.tags, { Purpose = "runtime-secret" })
}

resource "aws_secretsmanager_secret" "database_auth_token" {
  name = "vimcolorschemes/worker/database_auth_token"
  tags = merge(local.tags, { Purpose = "runtime-secret" })
}

resource "aws_secretsmanager_secret" "publish_webhook_url" {
  name = "vimcolorschemes/worker/publish_webhook_url"
  tags = merge(local.tags, { Purpose = "runtime-secret" })
}

resource "aws_iam_role_policy" "ecs_task_execution_secret_access" {
  name = "VimcolorschemesWorkerRuntimeSecretsRead"
  role = var.ecs_task_execution_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["secretsmanager:GetSecretValue"]
        Resource = [
          aws_secretsmanager_secret.github_token.arn,
          aws_secretsmanager_secret.database_url.arn,
          aws_secretsmanager_secret.database_auth_token.arn,
          aws_secretsmanager_secret.publish_webhook_url.arn,
        ]
      }
    ]
  })
}

resource "aws_cloudwatch_event_rule" "import" {
  name                = "import"
  description         = "Runs the vimcolorschemes import job"
  schedule_expression = "cron(0 13 * * ? *)"
  state               = "ENABLED"
  tags                = local.tags
}

resource "aws_cloudwatch_event_rule" "update" {
  name                = "update"
  description         = "Runs the vimcolorschemes update job"
  schedule_expression = "cron(30 13 * * ? *)"
  state               = "ENABLED"
  tags                = local.tags
}

resource "aws_cloudwatch_event_rule" "generate" {
  name                = "generate"
  description         = "Runs the vimcolorschemes generate job"
  schedule_expression = "cron(0 14 * * ? *)"
  state               = "ENABLED"
  tags                = local.tags
}

resource "aws_cloudwatch_event_rule" "publish" {
  name                = "publish"
  description         = "Runs the vimcolorschemes publish job"
  schedule_expression = "cron(30 14 * * ? *)"
  state               = "ENABLED"
  tags                = local.tags
}

resource "aws_cloudwatch_event_target" "import" {
  rule      = aws_cloudwatch_event_rule.import.name
  target_id = "import"
  arn       = aws_ecs_cluster.worker.arn
  role_arn  = local.ecs_events_role_arn
  input     = jsonencode({})

  ecs_target {
    task_count          = 1
    launch_type         = "FARGATE"
    platform_version    = "LATEST"
    task_definition_arn = var.bootstrap_task_definition_arn

    network_configuration {
      subnets          = var.default_subnet_ids
      security_groups  = var.import_security_group_ids
      assign_public_ip = true
    }
  }

  lifecycle {
    ignore_changes = [ecs_target[0].task_definition_arn]
  }
}

resource "aws_cloudwatch_event_target" "update" {
  rule      = aws_cloudwatch_event_rule.update.name
  target_id = "update"
  arn       = aws_ecs_cluster.worker.arn
  role_arn  = local.ecs_events_role_arn
  input = jsonencode({
    containerOverrides = [
      {
        name    = var.ecs_container_name
        command = ["update"]
      }
    ]
  })

  ecs_target {
    task_count          = 1
    launch_type         = "FARGATE"
    platform_version    = "LATEST"
    task_definition_arn = var.bootstrap_task_definition_arn

    network_configuration {
      subnets          = var.default_subnet_ids
      security_groups  = var.update_security_group_ids
      assign_public_ip = true
    }
  }

  lifecycle {
    ignore_changes = [ecs_target[0].task_definition_arn]
  }
}

resource "aws_cloudwatch_event_target" "generate" {
  rule      = aws_cloudwatch_event_rule.generate.name
  target_id = "generate"
  arn       = aws_ecs_cluster.worker.arn
  role_arn  = local.ecs_events_role_arn
  input = jsonencode({
    containerOverrides = [
      {
        name    = var.ecs_container_name
        command = ["generate"]
      }
    ]
  })

  ecs_target {
    task_count          = 1
    launch_type         = "FARGATE"
    platform_version    = "LATEST"
    task_definition_arn = var.bootstrap_task_definition_arn

    network_configuration {
      subnets          = var.default_subnet_ids
      security_groups  = var.generate_security_group_ids
      assign_public_ip = true
    }
  }

  lifecycle {
    ignore_changes = [ecs_target[0].task_definition_arn]
  }
}

resource "aws_cloudwatch_event_target" "publish" {
  rule      = aws_cloudwatch_event_rule.publish.name
  target_id = "publish"
  arn       = aws_ecs_cluster.worker.arn
  role_arn  = local.ecs_events_role_arn
  input = jsonencode({
    containerOverrides = [
      {
        name    = var.ecs_container_name
        command = ["publish"]
      }
    ]
  })

  ecs_target {
    task_count          = 1
    launch_type         = "FARGATE"
    platform_version    = "LATEST"
    task_definition_arn = var.bootstrap_task_definition_arn

    network_configuration {
      subnets          = var.default_subnet_ids
      security_groups  = var.publish_security_group_ids
      assign_public_ip = true
    }
  }

  lifecycle {
    ignore_changes = [ecs_target[0].task_definition_arn]
  }
}

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  tags            = merge(local.tags, { Purpose = "deployment" })
}

data "aws_iam_policy_document" "github_actions_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.github_repo}:ref:refs/heads/main"]
    }
  }
}

resource "aws_iam_role" "github_deploy" {
  name               = local.github_deploy_role_name
  description        = "GitHub Actions deploy role for vimcolorschemes worker"
  assume_role_policy = data.aws_iam_policy_document.github_actions_assume_role.json
  tags               = merge(local.tags, { Purpose = "deployment" })
}

resource "aws_iam_policy" "github_ecr_push" {
  name        = local.github_deploy_policy_ecr
  description = "Allow pushing vimcolorschemes worker image to ECR"
  tags        = merge(local.tags, { Purpose = "deployment" })

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:BatchGetImage",
          "ecr:CompleteLayerUpload",
          "ecr:DescribeImages",
          "ecr:DescribeRepositories",
          "ecr:GetDownloadUrlForLayer",
          "ecr:InitiateLayerUpload",
          "ecr:ListImages",
          "ecr:PutImage",
          "ecr:UploadLayerPart",
        ]
        Resource = aws_ecr_repository.worker.arn
      }
    ]
  })
}

resource "aws_iam_policy" "github_deploy_ecs_events" {
  name        = local.github_deploy_policy_schedule
  description = "Allow GitHub deploy workflow to update ECS task definition and EventBridge targets"
  tags        = merge(local.tags, { Purpose = "deployment" })

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "EcsTaskDefinitionDeploy"
        Effect   = "Allow"
        Action   = ["ecs:DescribeTaskDefinition", "ecs:RegisterTaskDefinition"]
        Resource = "*"
      },
      {
        Sid    = "UpdateWorkerSchedules"
        Effect = "Allow"
        Action = ["events:ListTargetsByRule", "events:PutTargets"]
        Resource = [
          aws_cloudwatch_event_rule.import.arn,
          aws_cloudwatch_event_rule.update.arn,
          aws_cloudwatch_event_rule.generate.arn,
          aws_cloudwatch_event_rule.publish.arn,
        ]
      },
      {
        Sid      = "PassEcsTaskRoles"
        Effect   = "Allow"
        Action   = ["iam:PassRole"]
        Resource = [local.ecs_task_execution_role_arn]
        Condition = {
          StringEquals = {
            "iam:PassedToService" = "ecs-tasks.amazonaws.com"
          }
        }
      },
      {
        Sid      = "PassEventsInvokeRole"
        Effect   = "Allow"
        Action   = ["iam:PassRole"]
        Resource = [local.ecs_events_role_arn]
        Condition = {
          StringEquals = {
            "iam:PassedToService" = "events.amazonaws.com"
          }
        }
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "github_deploy_ecr_attach" {
  role       = aws_iam_role.github_deploy.name
  policy_arn = aws_iam_policy.github_ecr_push.arn
}

resource "aws_iam_role_policy_attachment" "github_deploy_ecs_events_attach" {
  role       = aws_iam_role.github_deploy.name
  policy_arn = aws_iam_policy.github_deploy_ecs_events.arn
}

resource "aws_iam_user" "operator" {
  name = local.operator_user_name
  tags = merge(local.tags, { Purpose = "deployment" })
}

data "aws_iam_policy_document" "human_deployer_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${var.aws_account_id}:root"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:PrincipalArn"
      values   = [aws_iam_user.operator.arn]
    }
  }
}

resource "aws_iam_role" "human_deployer" {
  name               = local.human_deployer_role_name
  description        = "Human deploy role for vimcolorschemes worker"
  assume_role_policy = data.aws_iam_policy_document.human_deployer_assume_role.json
  tags               = merge(local.tags, { Purpose = "deployment" })
}

resource "aws_iam_role_policy_attachment" "human_deployer_ecr_attach" {
  role       = aws_iam_role.human_deployer.name
  policy_arn = aws_iam_policy.github_ecr_push.arn
}

resource "aws_iam_user_policy" "operator_assume_human_deployer" {
  name = "AssumeVimcolorschemesWorkerDeployerRole"
  user = aws_iam_user.operator.name
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "sts:AssumeRole"
        Resource = aws_iam_role.human_deployer.arn
      }
    ]
  })
}
