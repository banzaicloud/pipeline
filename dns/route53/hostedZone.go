// Copyright © 2018 Banzai Cloud
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

package route53

import (
	"fmt"
	"strings"
	"time"

	"emperror.dev/emperror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/pkg/amazon"
)

// createHostedZone creates a hosted zone on AWS Route53 with the given domain name
func (dns *awsRoute53) createHostedZone(domain string) (*route53.HostedZone, error) {
	log := loggerWithFields(logrus.Fields{"domain": domain})

	hostedZoneInput := &route53.CreateHostedZoneInput{
		CallerReference: aws.String(fmt.Sprintf("banzaicloud-pipepine-%d", time.Now().UnixNano())),
		Name:            aws.String(domain),
		HostedZoneConfig: &route53.HostedZoneConfig{
			Comment:     aws.String(createHostedZoneComment),
			PrivateZone: aws.Bool(false),
		},
	}

	hostedZoneOutput, err := dns.route53Svc.CreateHostedZone(hostedZoneInput)

	if err != nil {
		log.Errorf("creating Route53 hosted zone failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infof("route53 hosted zone created")

	return hostedZoneOutput.HostedZone, nil
}

// setHostedZoneSoaNTTL sets the NTTL value of the SOA record of the Hosted Zone identified by `id`
func (dns *awsRoute53) setHostedZoneSoaNTTL(id *string, nttl uint) error {
	const soa = "SOA"

	recordSetInput := &route53.ListResourceRecordSetsInput{HostedZoneId: id}
	recordSetOutput, err := dns.route53Svc.ListResourceRecordSets(recordSetInput)
	if err != nil {
		return emperror.WrapWith(err, "listing hosted zone record sets failed", "hostedZoneId", aws.StringValue(id))
	}

	var soaSet *route53.ResourceRecordSet
	for _, recordSet := range recordSetOutput.ResourceRecordSets {
		if aws.StringValue(recordSet.Type) == route53.RRTypeSoa {
			soaSet = recordSet
		}
	}
	if soaSet == nil {
		return errors.New("could not find SOA record")
	}

	for _, record := range soaSet.ResourceRecords {
		parts := strings.Split(*record.Value, " ")
		parts[len(parts)-1] = fmt.Sprintf("%d", nttl)
		*record.Value = strings.Join(parts, " ")
	}

	changeRecordSetInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: id,
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{Action: aws.String(route53.ChangeActionUpsert),
					ResourceRecordSet: soaSet,
				},
			},
		},
	}

	_, err = dns.route53Svc.ChangeResourceRecordSets(changeRecordSetInput)
	return emperror.Wrap(err, "changing SOA record set")
}

// getHostedZoneWithNameServers returns the hosted zone and it name servers with given id from AWS Route53
func (dns *awsRoute53) getHostedZoneWithNameServers(id *string) (*route53.GetHostedZoneOutput, error) {

	hostedZoneInput := &route53.GetHostedZoneInput{Id: id}
	hostedZoneOutput, err := dns.route53Svc.GetHostedZone(hostedZoneInput)
	if err != nil {
		return nil, err
	}

	return hostedZoneOutput, nil
}

// getHostedZone returns the hosted zone with given id from AWS Route53
func (dns *awsRoute53) getHostedZone(id *string) (*route53.HostedZone, error) {

	h, err := dns.getHostedZoneWithNameServers(id)
	if err != nil {
		return nil, err
	}

	return h.HostedZone, nil
}

// hostedZoneExistsByDomain returns hosted zone id if there is already a hosted zone created for the
// given domain in Route53. If there are multiple hosted zones registered for the domain
// that is considered an error
func (dns *awsRoute53) hostedZoneExistsByDomain(domain string) (string, error) {
	input := &route53.ListHostedZonesByNameInput{DNSName: aws.String(domain)}

	hostedZones, err := dns.route53Svc.ListHostedZonesByName(input)
	if err != nil {
		return "", err
	}

	var foundHostedZoneIds []string
	for _, hostedZone := range hostedZones.HostedZones {
		hostedZoneName := aws.StringValue(hostedZone.Name)
		hostedZoneName = hostedZoneName[:len(hostedZoneName)-1] // remove trailing '.' from name

		if hostedZoneName == domain {
			foundHostedZoneIds = append(foundHostedZoneIds, aws.StringValue(hostedZone.Id))
		}
	}

	if len(foundHostedZoneIds) > 1 {
		return "", fmt.Errorf("multiple hosted zones %v found for domain '%s'", foundHostedZoneIds, domain)
	}

	if len(foundHostedZoneIds) == 0 {
		return "", nil
	}

	return foundHostedZoneIds[0], nil
}

// deleteHostedZone deletes the hosted zone with the given id from AWS Route53
func (dns *awsRoute53) deleteHostedZone(id *string) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(id)})

	listResourceRecordSetsInput := &route53.ListResourceRecordSetsInput{HostedZoneId: id}
	resourceRecordSets, err := dns.route53Svc.ListResourceRecordSets(listResourceRecordSetsInput)
	if err != nil {
		log.Errorf("retrieving resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
		return err
	}

	var resourceRecordSetChanges []*route53.ResourceRecordSet

	for _, resourceRecordSet := range resourceRecordSets.ResourceRecordSets {
		if aws.StringValue(resourceRecordSet.Type) != route53.RRTypeNs && aws.StringValue(resourceRecordSet.Type) != route53.RRTypeSoa {
			resourceRecordSetChanges = append(resourceRecordSetChanges, resourceRecordSet)
		}
	}

	if len(resourceRecordSetChanges) > 0 {
		err = dns.deleteResourceRecordSets(id, resourceRecordSetChanges)
		if err != nil {
			log.Errorf("deleting all resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
			return err
		}
	}

	hostedZoneInput := &route53.DeleteHostedZoneInput{Id: id}

	_, err = dns.route53Svc.DeleteHostedZone(hostedZoneInput)
	if err != nil {
		log.Errorf("deleting hosted zone failed: %s", extractErrorMessage(err))
	}
	log.Infof("hosted zone deleted")

	return err
}

// deleteHostedZoneResourceRecordSetsOwnedBy deletes resource records set of hosted zone and belong to the owner of the given id.
func (dns *awsRoute53) deleteHostedZoneResourceRecordSetsOwnedBy(hostedZoneId *string, ownerId string) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(hostedZoneId), "ownerId": ownerId})

	listResourceRecordSetsInput := &route53.ListResourceRecordSetsInput{HostedZoneId: hostedZoneId}
	resourceRecordSets, err := dns.route53Svc.ListResourceRecordSets(listResourceRecordSetsInput)
	if err != nil {
		log.Errorf("retrieving resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
		return err
	}

	ownerReference := "external-dns/owner=" + ownerId

	var ownedRecordNames = make(map[string]bool)

	for _, resourceRecordSet := range resourceRecordSets.ResourceRecordSets {
		if aws.StringValue(resourceRecordSet.Type) == route53.RRTypeTxt {
			for _, resourceRecord := range resourceRecordSet.ResourceRecords {
				if strings.Contains(aws.StringValue(resourceRecord.Value), ownerReference) {
					ownedRecordNames[aws.StringValue(resourceRecordSet.Name)] = true
					break
				}
			}
		}
	}

	var resourceRecordSetChanges []*route53.ResourceRecordSet
	for _, resourceRecordSet := range resourceRecordSets.ResourceRecordSets {
		if aws.StringValue(resourceRecordSet.Type) != route53.RRTypeNs && aws.StringValue(resourceRecordSet.Type) != route53.RRTypeSoa {
			if _, ok := ownedRecordNames[aws.StringValue(resourceRecordSet.Name)]; ok {
				resourceRecordSetChanges = append(resourceRecordSetChanges, resourceRecordSet)
			}
		}
	}

	if len(resourceRecordSetChanges) > 0 {
		err = dns.deleteResourceRecordSets(hostedZoneId, resourceRecordSetChanges)
		if err != nil {
			log.Errorf("deleting resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
			return err
		}
	}

	return nil
}

// setHostedZoneAuthorisation sets up authorisation for the Route53 hosted zone identified by the specified id.
// It creates a policy that allows changing only the specified hosted zone and a IAM user with the policy attached.
func (dns *awsRoute53) setHostedZoneAuthorisation(hostedZoneId string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"hostedzone": hostedZoneId})

	var policy *iam.Policy
	var err error

	if len(ctx.state.policyArn) > 0 {
		policy, err = amazon.GetPolicy(dns.iamSvc, ctx.state.policyArn)
		if err != nil {
			log.Errorf("retrieving route53 policy '%s' failed: %s", ctx.state.policyArn, extractErrorMessage(err))
			return err
		}
	}

	if policy == nil {
		// create route53 policy
		policy, err = dns.createHostedZoneRoute53Policy(ctx.state.organisationId, hostedZoneId)
		if err != nil {
			return err
		}
	} else {
		log.Infof("skip creating route53 policy for hosted zone as it already exists: arn='%s'", ctx.state.policyArn)
	}

	ctx.registerRollback(func() error {
		return dns.deletePolicy(policy.Arn)
	})

	if ctx.state.policyArn != aws.StringValue(policy.Arn) {
		ctx.state.policyArn = aws.StringValue(policy.Arn)
		if err = dns.stateStore.update(ctx.state); err != nil {
			log.Errorf("failed to update state store: %s", extractErrorMessage(err))
			return err
		}
	}

	// create IAM user
	org, err := dns.getOrganization(ctx.state.organisationId)
	if err != nil {
		log.Errorf("retrieving organization with id %d failed: %s", ctx.state.organisationId, extractErrorMessage(err))
		return err
	}

	userName := aws.String(getIAMUserName(org))
	err = dns.createHostedZoneIAMUser(userName, aws.String(ctx.state.policyArn), ctx)
	if err != nil {
		log.Errorf("setting up IAM user '%s' for hosted zone failed: %s", aws.StringValue(userName), extractErrorMessage(err))
		return err
	}
	log.Info("IAM user for hosted zone has been set up")

	return nil
}

// createHostedZoneIAMUser creates a IAM user and attaches the route53 policy identified by the given arn
func (dns *awsRoute53) createHostedZoneIAMUser(userName, route53PolicyArn *string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"IAMUser": aws.StringValue(userName), "policy": aws.StringValue(route53PolicyArn)})

	iamUser, err := dns.getIAMUser(userName)
	if err != nil {
		return err
	}

	if iamUser == nil {
		// create IAM User
		iamUser, err = dns.createIAMUser(userName)
		if err != nil {
			return err
		}
	}

	ctx.registerRollback(func() error {
		return dns.deleteIAMUser(iamUser.UserName)
	})

	if ctx.state.iamUser != aws.StringValue(iamUser.UserName) {
		ctx.state.iamUser = aws.StringValue(iamUser.UserName)

		if err := dns.stateStore.update(ctx.state); err != nil {
			return err
		}
	} else {
		log.Info("skip creating IAM user as it already exists")
	}

	// attach policy to user

	// check is the IAM user already has this policy attached
	policyAlreadyAttached, err := amazon.IsUserPolicyAttached(dns.iamSvc, userName, route53PolicyArn)
	if err != nil {
		return err
	}

	if !policyAlreadyAttached {
		if err := dns.attachUserPolicy(aws.String(ctx.state.iamUser), route53PolicyArn); err != nil {
			return err
		}
	} else {
		log.Info("skip attaching policy to user as it is already attached")
	}

	ctx.registerRollback(func() error {
		return dns.detachUserPolicy(aws.String(ctx.state.iamUser), route53PolicyArn)
	})

	// setup Amazon access keys for IAM usser
	err = dns.setupAmazonAccess(aws.StringValue(userName), ctx)

	if err != nil {
		log.Errorf("setting up Amazon access key for user failed: %s", extractErrorMessage(err))
		return err
	}

	return nil
}

// chainToBaseDomain chains the given hosted zone representing a domain into
// the hosted zone that corresponds to the parent base domain
func (dns *awsRoute53) chainToBaseDomain(hostedZoneId string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": hostedZoneId})

	hostedZone, err := dns.getHostedZoneWithNameServers(aws.String(hostedZoneId))
	if err != nil {
		return err
	}

	resourceRecordSet, err := dns.getResourceRecordSetFromBaseHostedZone(hostedZone.HostedZone.Name)
	if err != nil {
		log.Errorf("getting resource record set from base hosted zone failed: %s", extractErrorMessage(err))
		return err
	}

	if resourceRecordSet != nil {
		// domain already linked to parent base domain. verify if NS resource records is in sync, if not update them
		if nameServerMatch(hostedZone.DelegationSet, resourceRecordSet) {
			log.Infoln("skip linking hosted zone to base hosted zone as it's already done !")
			return nil
		}

		// update NS resource record set entry in base domain
		resourceRecordSet.ResourceRecords = createResourceRecordsFromDelegationSet(hostedZone.DelegationSet)
		err := dns.updateResourceRecordSets(aws.String(dns.baseHostedZoneId), []*route53.ResourceRecordSet{resourceRecordSet})
		if err != nil {
			return err
		}
	} else {
		// domain not linked to base domain yet. Link it to parent
		resourceRecordSet := &route53.ResourceRecordSet{
			Name:            hostedZone.HostedZone.Name,
			Type:            aws.String(route53.RRTypeNs),
			ResourceRecords: createResourceRecordsFromDelegationSet(hostedZone.DelegationSet),
			TTL:             aws.Int64(300),
		}

		err := dns.createResourceRecordSets(aws.String(dns.baseHostedZoneId), []*route53.ResourceRecordSet{resourceRecordSet})
		if err != nil {
			return err
		}

		// register rollback function
		ctx.registerRollback(func() error {
			return dns.deleteResourceRecordSets(aws.String(dns.baseHostedZoneId), []*route53.ResourceRecordSet{resourceRecordSet})
		})
	}
	return nil
}

// unChainFromBaseDomain removes the ResourceRecordSet that corresponds to the passed domain from parent base hosted zone
func (dns *awsRoute53) unChainFromBaseDomain(domain string) error {
	log := loggerWithFields(logrus.Fields{"domain": domain})

	log.Infoln("removing domain from base domain")

	if !strings.HasSuffix(domain, ".") {
		domain += "."

	}
	resourceRecordSet, err := dns.getResourceRecordSetFromBaseHostedZone(aws.String(domain))
	if err != nil {
		log.Errorf("getting resource record set from base hosted zone failed: %s", extractErrorMessage(err))
		return err
	}

	if resourceRecordSet != nil {
		return dns.deleteResourceRecordSets(aws.String(dns.baseHostedZoneId), []*route53.ResourceRecordSet{resourceRecordSet})
	}

	log.Infoln("skip removing domain from base domain as it's been already removed")
	return nil
}

// getResourceRecordSetFromBaseHostedZone retrieves the NS type ResourceRecordSet that corresponds to the given record set name from base hosted zone
// If none ResourceRecordSet found returns nil
func (dns *awsRoute53) getResourceRecordSetFromBaseHostedZone(name *string) (*route53.ResourceRecordSet, error) {
	baseHostedZone, err := dns.getHostedZone(aws.String(dns.baseHostedZoneId))
	if err != nil {
		return nil, err
	}

	listResourceRecordSets := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    baseHostedZone.Id,
		StartRecordType: aws.String(route53.RRTypeNs),
		StartRecordName: name,
		MaxItems:        aws.String("1"),
	}
	res, err := dns.route53Svc.ListResourceRecordSets(listResourceRecordSets)
	if err != nil {
		return nil, err
	}

	if len(res.ResourceRecordSets) > 0 && (aws.StringValue(res.ResourceRecordSets[0].Name) == aws.StringValue(name)) {
		return res.ResourceRecordSets[0], nil
	}

	return nil, nil
}

// createResourceRecordSets creates a ResourceRecordSets in the hosted zone with the given id in Route53 service
func (dns *awsRoute53) createResourceRecordSets(zoneId *string, rrs []*route53.ResourceRecordSet) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(zoneId)})

	log.Infoln("adding resource record sets")
	return dns.changeResourceRecordSet(aws.String(route53.ChangeActionCreate), zoneId, rrs)
}

// updateResourceRecordSets updates a ResourceRecordSets of a hosted zone with the given id in Route53 service
func (dns *awsRoute53) updateResourceRecordSets(zoneId *string, rrs []*route53.ResourceRecordSet) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(zoneId)})

	log.Infoln("updating resource record sets")
	return dns.changeResourceRecordSet(aws.String(route53.ChangeActionUpsert), zoneId, rrs)
}

// deleteResourceRecordSets deletes the ResourceRecordSets of a hosted zone with the given id in Route53 service
func (dns *awsRoute53) deleteResourceRecordSets(zoneId *string, rrs []*route53.ResourceRecordSet) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(zoneId)})

	log.Infoln("deleting resource record sets")
	return dns.changeResourceRecordSet(aws.String(route53.ChangeActionDelete), zoneId, rrs)
}

// changeResourceRecordSets executes the ChangeAction on the given ResourceRecordSets of a hosted zone
func (dns *awsRoute53) changeResourceRecordSet(action, zoneId *string, rrs []*route53.ResourceRecordSet) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(zoneId), "action": aws.StringValue(action)})

	log.Infoln("executing action on resource record sets")
	var changes []*route53.Change

	if len(rrs) == 0 {
		return nil // nop
	}

	for _, r := range rrs {
		changes = append(changes, &route53.Change{
			Action:            action,
			ResourceRecordSet: r,
		})
	}

	changeInput := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: zoneId,
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
	}

	changeOutput, err := dns.route53Svc.ChangeResourceRecordSets(changeInput)
	if err != nil {
		return err
	}

	log.Infoln("wait until resource record sets changed")
	err = dns.route53Svc.WaitUntilResourceRecordSetsChanged(
		&route53.GetChangeInput{
			Id: changeOutput.ChangeInfo.Id,
		})

	if err != nil {
		return err
	}

	return nil
}

// nameServerMatch returns true if the name servers of the delegation set matches the
// resource records in the provided resource records set, otherwise returns false
func nameServerMatch(ds *route53.DelegationSet, rrs *route53.ResourceRecordSet) bool {
	if aws.StringValue(rrs.Type) != route53.RRTypeNs {
		return false // the resource record set must be of type NameServer
	}

	if len(ds.NameServers) != len(rrs.ResourceRecords) {
		return false
	}

	for _, rr := range rrs.ResourceRecords {
		var i int
		for i = 0; i < len(ds.NameServers); i++ {
			if aws.StringValue(rr.Value) == aws.StringValue(ds.NameServers[i]) {
				break
			}
		}

		if i == len(ds.NameServers) {
			return false
		}
	}

	return true
}

func createResourceRecordsFromDelegationSet(ds *route53.DelegationSet) []*route53.ResourceRecord {
	var rr []*route53.ResourceRecord
	for _, nameServer := range ds.NameServers {
		rr = append(rr, &route53.ResourceRecord{Value: nameServer})
	}

	return rr
}

func stripHostedZoneId(id string) string {
	return strings.Replace(id, "/hostedzone/", "", 1)
}
