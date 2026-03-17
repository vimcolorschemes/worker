variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "aws_profile" {
  type    = string
  default = ""
}

variable "aws_account_id" {
  type    = string
  default = "123456789012"
}

variable "project" {
  type    = string
  default = "vimcolorschemes"
}

variable "service" {
  type    = string
  default = "worker"
}

variable "environment" {
  type    = string
  default = "production"
}

variable "owner" {
  type    = string
  default = "owner"
}

variable "managed_by" {
  type    = string
  default = "terraform"
}

variable "github_repo" {
  type    = string
  default = "vimcolorschemes/worker"
}

variable "ecr_repository_name" {
  type    = string
  default = "vimcolorschemes/worker"
}

variable "ecs_cluster_name" {
  type    = string
  default = "vimcolorschemes-worker"
}

variable "ecs_task_family" {
  type    = string
  default = "run-job"
}

variable "ecs_container_name" {
  type    = string
  default = "vimcolorschemes-worker"
}

variable "ecs_task_execution_role_name" {
  type    = string
  default = "ecsTaskExecutionRole"
}

variable "ecs_events_role_name" {
  type    = string
  default = "ecsEventsRole"
}

variable "cloudwatch_log_group_name" {
  type    = string
  default = "/ecs/run-job"
}

variable "default_subnet_ids" {
  type    = list(string)
  default = ["subnet-xxxxxxxxxxxxxxxxx"]
}

variable "import_security_group_ids" {
  type    = list(string)
  default = ["sg-xxxxxxxxxxxxxxxxx"]
}

variable "update_security_group_ids" {
  type    = list(string)
  default = ["sg-yyyyyyyyyyyyyyyyy"]
}

variable "generate_security_group_ids" {
  type    = list(string)
  default = ["sg-zzzzzzzzzzzzzzzzz"]
}

variable "bootstrap_task_definition_arn" {
  type    = string
  default = "arn:aws:ecs:us-east-1:123456789012:task-definition/run-job:1"
}
