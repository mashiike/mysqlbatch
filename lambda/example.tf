
resource "aws_iam_role" "mysqlbatch" {
  name = "mysqlbatch_lambda"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "mysqlbatch" {
  role       = aws_iam_role.mysqlbatch.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_iam_role_policy_attachment" "mysqlbatch_ssm" {
  role       = aws_iam_role.mysqlbatch.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMReadOnlyAccess"
}


data "archive_file" "mysqlbatch_dummy" {
  type        = "zip"
  output_path = "${path.module}/mysqlbatch_dummy.zip"
  source {
    content  = "mysqlbatch_dummy"
    filename = "bootstrap"
  }
  depends_on = [
    null_resource.mysqlbatch_dummy
  ]
}

resource "null_resource" "mysqlbatch_dummy" {}

resource "aws_lambda_function" "mysqlbatch" {
  lifecycle {
    ignore_changes = all
  }

  function_name = "mysqlbatch"
  role          = aws_iam_role.mysqlbatch.arn

  handler = "bootstrap"
  runtime = "provided.al2"
  vpc_config {
    subnet_ids         = data.aws_subnets.default.ids
    security_group_ids = [data.aws_security_group.internal.id]
  }
  filename = data.archive_file.mysqlbatch_dummy.output_path
}

resource "aws_ssm_parameter" "DBPASSWORD" {
  name        = "/mysqlbatch/DBPASSWORD"
  description = "mysqlbatch db password"
  type        = "SecureString"
  value       = local.db_password
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [local.vpc_id]
  }
}

data "aws_security_group" "internal" {
  id = local.security_group_id
}

data "aws_rds_cluster" "mysql" {
  cluster_identifier = local.cluster_identifier
}
