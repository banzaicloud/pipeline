// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package arn collects supporting entities related to working with Amazon AWS
// resource names.
//
// ARN represents an Amazon resource name uniquely identifying an AWS resource.
// Unfortunately the aws-sdk-go/aws/arn.ARN type stores the Resource section as
// a single string which is not detailed enough for some use cases, this type
// implements a finer detailed interface for ARNs.
//
// Some example ARNs:
// arn:aws:elasticbeanstalk:us-east-1:123456789012:environment/My App/MyEnvironment
// arn:aws:iam::123456789012:user/David
// arn:aws:rds:eu-west-1:123456789012:db:mysql-db
// arn:aws:s3:::my_corporate_bucket/exampleobject.png
package arn

import (
	"strings"

	"emperror.dev/errors"
)

const (
	// ARNPathSeparator is the separating character between the resource path
	// elements and also is the leading and trailing character of the resource
	// path.
	ARNPathSeparator = "/"

	// ARNQualifierSeparator is the separating character between the resource ID
	// and qualifier.
	ARNQualifierSeparator = ":"

	// ARNSectionPrefix is the prefix section of an ARN string representation.
	ARNSectionPrefix = "arn"

	// ARNSections describes the required sections of an ARN.
	ARNSections = "prefix" + ARNSectionSeparator +
		"partition" + ARNSectionSeparator +
		"service" + ARNSectionSeparator +
		"region" + ARNSectionSeparator +
		"accountID" + ARNSectionSeparator +
		"resource"

	// ARNSectionSeparator is the character separating ARN sections in an ARN
	// string representation.
	ARNSectionSeparator = ":"

	// ARNTypeSeparators are the separator characters between the optional
	// resource type and the follwing sub-section (optional path or required
	// name).
	ARNTypeSeparators = ARNSectionSeparator + ARNPathSeparator
)

var (
	// ErrorInvalidPrefix is returned when the ARN prefix is not the expected
	// value.
	ErrorInvalidPrefix = errors.New("invalid ARN prefix, 'arn:' is expected")

	// ErrorInvalidStructure is returned when the ARN section structure does not
	// meet the section requiremens and required sections are missing.
	ErrorInvalidStructure = errors.New("invalid ARN section structure, 6 sections are expected")
)

// AccountID returns the content of the corresponding section.
//
// The ID of the AWS account that owns the resource, without the hyphens. For
// example, 123456789012. Note that the ARNs for some resources don't require an
// account number, so this component might be omitted.
func AccountID(arn string) (accountID string) {
	if !IsARN(arn) {
		return ""
	}

	// Note: validation ensures len(split) >= len(strings.Split(ARNSections,
	// ARNSectionSeparator)).
	return strings.SplitN(arn, ARNSectionSeparator, len(strings.Split(ARNSections, ARNSectionSeparator)))[4]
}

// IsARN determines whether a string represents a valid ARN object by checking
// the ARN schema of the string.
func IsARN(candidate string) (isARN bool) {
	return ValidateARN(candidate) == nil
}

// NewARN creates an ARN string from the specified sections and sub-sections.
func NewARN(
	partition, service, region, accountID, resourceType, resourcePathOrParent, resourceName, resourceQualifier string,
) (arn string) {
	builder := strings.Builder{}
	builder.WriteString(ARNSectionPrefix)
	builder.WriteString(ARNSectionSeparator)
	builder.WriteString(partition)
	builder.WriteString(ARNSectionSeparator)
	builder.WriteString(service)
	builder.WriteString(ARNSectionSeparator)
	builder.WriteString(region)
	builder.WriteString(ARNSectionSeparator)
	builder.WriteString(accountID)
	builder.WriteString(ARNSectionSeparator)

	if resourceType != "" {
		builder.WriteString(resourceType)
	}

	builder.WriteString(resourcePathOrParent)
	builder.WriteString(resourceName)

	if resourceQualifier != "" {
		builder.WriteString(ARNQualifierSeparator)
		builder.WriteString(resourceQualifier)
	}

	return builder.String()
}

// Partition returns the content of the corresponding section.
//
// The partition that the resource is in. For standard AWS regions, the
// partition is "aws". If you have resources in other partitions, the partition
// is "aws-partitionname". For example, the partition for resources in the China
// (Beijing) region is "aws-cn".
func Partition(arn string) (partition string) {
	if !IsARN(arn) {
		return ""
	}

	// Note: validation ensures len(split) >= len(strings.Split(ARNSections,
	// ARNSectionSeparator)).
	return strings.SplitN(arn, ARNSectionSeparator, len(strings.Split(ARNSections, ARNSectionSeparator)))[1]
}

// Region returns the content of the corresponding section.
//
// The region the resource resides in. Note that the ARNs for some resources do
// not require a region, so this component might be omitted.
func Region(arn string) (region string) {
	if !IsARN(arn) {
		return ""
	}

	// Note: validation ensures len(split) >= len(strings.Split(ARNSections,
	// ARNSectionSeparator)).
	return strings.SplitN(arn, ARNSectionSeparator, len(strings.Split(ARNSections, ARNSectionSeparator)))[3]
}

// Resource returns the content of the corresponding section.
//
// An example resource looks like this:
//
// type(:|/)(parent-resource/|path/elements/)id:qualifier
//
// The content of this part of the ARN varies by service. It often includes an
// indicator of the type of resource — for example, an IAM user or Amazon RDS
// database - followed by a slash (/) or a colon (:), followed by the resource
// name itself. Some services allow paths for resource names, as described in
// http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#arns-paths.
func Resource(arn string) (resource string) {
	if !IsARN(arn) {
		return ""
	}

	// Note: validation ensures len(split) >= len(strings.Split(ARNSections,
	// ARNSectionSeparator)).
	return strings.SplitN(arn, ARNSectionSeparator, len(strings.Split(ARNSections, ARNSectionSeparator)))[5]
}

// ResourceName returns the required subsection of the ARN resource representing
// the name of the identified resource.
//
// WARNING: some services which incorporate colons or slashes in resource names
// might define type counter-intuitively, for example S3 has the bucket name at
// the resource type sub-section, the intermediate path in the resource path or
// parent and the object name at the resource name.
func ResourceName(arn string) (resourceName string) {
	if !IsARN(arn) {
		return ""
	}

	qualifier := ResourceQualifier(arn)
	qualifierlessResource := Resource(arn)
	if qualifier != "" {
		qualifierlessResource = strings.TrimSuffix(qualifierlessResource, ARNQualifierSeparator+qualifier)
	}

	nameSeparatorIndex := strings.LastIndexAny(qualifierlessResource, ARNTypeSeparators)
	if nameSeparatorIndex == -1 { // Note: name ends in separator?
		return qualifierlessResource
	}

	return qualifierlessResource[nameSeparatorIndex+1:] // Note: len(slice) begin index results in empty string.
}

// ResourcePathOrParent returns the optional subsection of the ARN resource
// representing the path or parent of the identified resource.
//
// WARNING: some services which incorporate colons or slashes in resource names
// might define type counter-intuitively, for example S3 has the bucket name at
// the resource type sub-section, the intermediate path in the resource path or
// parent and the object name at the resource name.
func ResourcePathOrParent(arn string) (resourcePathOrParent string) {
	if !IsARN(arn) {
		return ""
	}

	qualifier := ResourceQualifier(arn)
	typeAndPath := strings.TrimSuffix(Resource(arn), qualifier)
	if qualifier != "" {
		typeAndPath = strings.TrimSuffix(typeAndPath, ARNQualifierSeparator)
	}
	typeAndPath = strings.TrimSuffix(typeAndPath, ResourceName(arn))

	return strings.TrimPrefix(typeAndPath, ResourceType(arn))
}

// ResourceName returns the optional subsection of the ARN.Resource qualifying
// the resource ID.
func ResourceQualifier(arn string) (resourceQualifier string) {
	if !IsARN(arn) {
		return ""
	}

	resource := Resource(arn)
	qualifierSeparatorIndex := strings.LastIndex(resource, ARNQualifierSeparator)
	typeSeparatorIndex := strings.IndexAny(resource, ARNTypeSeparators)
	if qualifierSeparatorIndex == -1 ||
		typeSeparatorIndex == -1 ||
		qualifierSeparatorIndex <= typeSeparatorIndex {
		return ""
	}

	// Note: there is a : after a : or / which means it's a qualifier separator.
	return resource[qualifierSeparatorIndex+1:]
}

// ResourceType returns the optional subsection of the ARN resource representing
// the kind of the identified resource.
//
// WARNING: some services which incorporate colons or slashes in resource names
// might define type counter-intuitively, for example S3 has the bucket name at
// the resource type sub-section, the intermediate path in the resource path or
// parent and the object name at the resource name.
func ResourceType(arn string) (resourceName string) {
	if !IsARN(arn) {
		return ""
	}

	resource := Resource(arn)
	typeEndIndex := strings.IndexAny(resource, ARNTypeSeparators)
	if typeEndIndex == -1 {
		return ""
	}

	return resource[:typeEndIndex]
}

// Service returns the content of the corresponding section.
//
// The service namespace that identifies the AWS product (for example, Amazon
// S3, IAM, or Amazon RDS). For a list of namespaces, see
// http://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html#genref-aws-service-namespaces.
func Service(arn string) (service string) {
	if !IsARN(arn) {
		return ""
	}

	// Note: validation ensures len(split) >= len(strings.Split(ARNSections,
	// ARNSectionSeparator)).
	return strings.SplitN(arn, ARNSectionSeparator, len(strings.Split(ARNSections, ARNSectionSeparator)))[2]
}

// ValidateARN returns an error in case the specified string representation is
// not a valid ARN.
func ValidateARN(candidate string) (err error) {
	if !strings.HasPrefix(candidate, ARNSectionPrefix+ARNSectionSeparator) {
		return errors.WithDetails(ErrorInvalidPrefix, "candidate", candidate)
	}

	if strings.Count(candidate, ARNSectionSeparator) < strings.Count(ARNSections, ARNSectionSeparator) {
		return errors.WithDetails(ErrorInvalidStructure, "candidate", candidate)
	}

	return nil
}
