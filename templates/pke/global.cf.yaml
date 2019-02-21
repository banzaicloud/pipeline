AWSTemplateFormatVersion: 2010-09-09
Description: 'Role(s) and InstanceProfile(s) for Banzai Cloud Pipeline Kubernetes Engine'
Resources:
  WorkerRole:
    Type: AWS::IAM::Role
    Properties:
      Path: /
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
        -
          Effect: "Allow"
          Principal:
            Service:
            - "ec2.amazonaws.com"
          Action:
          - "sts:AssumeRole"
      RoleName: !Join ["", [ !Ref "AWS::StackName" , "-worker" ]]
      Policies:
      - PolicyName: PkeKubernetesMasterPolicy
        PolicyDocument:
          Version: "2012-10-17"
          Statement:
            - Action: [
              'ec2:Describe*',
              'ecr:GetAuthorizationToken',
              'ecr:BatchCheckLayerAvailability',
              'ecr:GetDownloadUrlForLayer',
              'ecr:GetRepositoryPolicy',
              'ecr:DescribeRepositories',
              'ecr:ListImages',
              'ecr:BatchGetImage',
              'autoscaling:DescribeAutoScalingGroups',
              'autoscaling:UpdateAutoScalingGroup',
              'autoscaling:DescribeAutoScalingInstances',
              'autoscaling:DescribeTags',
              'autoscaling:DescribeLaunchConfigurations',
              'autoscaling:SetDesiredCapacity',
              'autoscaling:TerminateInstanceInAutoScalingGroup',
              'autoscaling:PutLifecycleHook',
              'autoscaling:RecordLifecycleActionHeartbeat',
              'autoscaling:DescribeLifecycleHooks',
              'autoscaling:CompleteLifecycleAction',
              'autoscaling:DeleteLifecycleHook',
              'autoscaling:DetachInstances',
              ]
              Effect: Allow
              Resource: '*'

  MasterRole:
    Type: AWS::IAM::Role
    Properties:
      Path: /
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
        -
          Effect: "Allow"
          Principal:
            Service:
            - "ec2.amazonaws.com"
          Action:
          - "sts:AssumeRole"
      RoleName: !Join ["", [ !Ref "AWS::StackName" , "-master" ]]
      Policies:
      - PolicyName: PkeKubernetesWorkerPolicy
        PolicyDocument:
          Version: "2012-10-17"
          Statement:
          - Action: [
            'autoscaling:DescribeAutoScalingGroups',
            'autoscaling:DescribeLaunchConfigurations',
            'autoscaling:DescribeTags',
            'ec2:DescribeInstances',
            'ec2:DescribeRegions',
            'ec2:DescribeRouteTables',
            'ec2:DescribeSecurityGroups',
            'ec2:DescribeSubnets',
            'ec2:DescribeVolumes',
            'ec2:CreateSecurityGroup',
            'ec2:CreateTags',
            'ec2:CreateVolume',
            'ec2:ModifyInstanceAttribute',
            'ec2:ModifyVolume',
            'ec2:AttachVolume',
            'ec2:AuthorizeSecurityGroupIngress',
            'ec2:CreateRoute',
            'ec2:DeleteRoute',
            'ec2:DeleteSecurityGroup',
            'ec2:DeleteVolume',
            'ec2:DetachVolume',
            'ec2:RevokeSecurityGroupIngress',
            'ec2:DescribeVpcs',
            'elasticloadbalancing:AddTags',
            'elasticloadbalancing:AttachLoadBalancerToSubnets',
            'elasticloadbalancing:ApplySecurityGroupsToLoadBalancer',
            'elasticloadbalancing:CreateLoadBalancer',
            'elasticloadbalancing:CreateLoadBalancerPolicy',
            'elasticloadbalancing:CreateLoadBalancerListeners',
            'elasticloadbalancing:ConfigureHealthCheck',
            'elasticloadbalancing:DeleteLoadBalancer',
            'elasticloadbalancing:DeleteLoadBalancerListeners',
            'elasticloadbalancing:DescribeLoadBalancers',
            'elasticloadbalancing:DescribeLoadBalancerAttributes',
            'elasticloadbalancing:DetachLoadBalancerFromSubnets',
            'elasticloadbalancing:DeregisterInstancesFromLoadBalancer',
            'elasticloadbalancing:ModifyLoadBalancerAttributes',
            'elasticloadbalancing:RegisterInstancesWithLoadBalancer',
            'elasticloadbalancing:SetLoadBalancerPoliciesForBackendServer',
            'elasticloadbalancing:AddTags',
            'elasticloadbalancing:CreateListener',
            'elasticloadbalancing:CreateTargetGroup',
            'elasticloadbalancing:DeleteListener',
            'elasticloadbalancing:DeleteTargetGroup',
            'elasticloadbalancing:DescribeListeners',
            'elasticloadbalancing:DescribeLoadBalancerPolicies',
            'elasticloadbalancing:DescribeTargetGroups',
            'elasticloadbalancing:DescribeTargetHealth',
            'elasticloadbalancing:ModifyListener',
            'elasticloadbalancing:ModifyTargetGroup',
            'elasticloadbalancing:RegisterTargets',
            'elasticloadbalancing:SetLoadBalancerPoliciesOfListener',
            'iam:CreateServiceLinkedRole',
            ]
            Effect: Allow
            Resource: '*'
  MasterInstanceProfile:
    Type: 'AWS::IAM::InstanceProfile'
    Properties:
      Path: /
      Roles:
      - !Ref MasterRole
      InstanceProfileName: !Join [ "", [ !Ref "AWS::StackName", "-", "master", "-", "profile" ] ]
  WorkerInstanceProfile:
    Type: 'AWS::IAM::InstanceProfile'
    Properties:
      Path: /
      Roles:
      - !Ref WorkerRole
      InstanceProfileName: !Join [ "", [ !Ref "AWS::StackName", "-", "worker", "-", "profile" ] ]

Outputs:
  StackName:
    Description: 'Stack name'
    Value: !Sub '${AWS::StackName}'
  MasterInstanceProfile:
    Description: 'Kubernetes Master Instance Profile'
    Value: !Ref MasterInstanceProfile
    Export:
      Name: !Sub '${AWS::StackName}-MasterInstanceProfile'
  WorkerInstanceProfile:
    Description: 'Kubernetes Worker Instance Profile'
    Value: !Ref WorkerInstanceProfile
    Export:
      Name: !Sub '${AWS::StackName}-WorkerInstanceProfile'
  WorkerRole:
    Description: 'Kubernetes Worker Role'
    Value: !Ref WorkerRole
    Export:
      Name: !Sub '${AWS::StackName}-WorkerRole'
  MasterRole:
    Description: 'Kubernetes Master Role'
    Value: !Ref MasterRole
    Export:
      Name: !Sub '${AWS::StackName}-MasterRole'