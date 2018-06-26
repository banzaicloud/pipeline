package route53

import (
	"fmt"
	"github.com/banzaicloud/pipeline/dns/route53/model"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster"
)

// awsRoute53DatabaseStateStore is a database backed state store for
// managing the state of the domains registered by us in the Amazon Route53 external DNS service
type awsRoute53DatabaseStateStore struct{}

// create persists the given domain state to database
func (stateStore *awsRoute53DatabaseStateStore) create(state *domainState) error {
	db := model.GetDB()

	rec := createRoute53Domain(state)

	return db.Create(rec).Error
}

// update persists the changes of given domain state to database
func (stateStore *awsRoute53DatabaseStateStore) update(state *domainState) error {
	db := model.GetDB()

	dbRec := &route53model.Route53Domain{}
	err := db.Where(&route53model.Route53Domain{OrganizationId: state.organisationId, Domain: state.domain}).First(dbRec).Error
	if err != nil {
		return err
	}

	dbRec.Status = state.status
	dbRec.PolicyArn = state.policyArn
	dbRec.HostedZoneId = state.hostedZoneId
	dbRec.IamUser = state.iamUser
	dbRec.AwsAccessKeyId = state.awsAccessKeyId
	dbRec.ErrorMessage = state.errMsg

	return db.Save(dbRec).Error
}

// find looks up in the database the domain state identified by origId and domain. The found data is passed back
// through stateOut
func (stateStore *awsRoute53DatabaseStateStore) find(orgId uint, domain string, stateOut *domainState) (bool, error) {
	db := model.GetDB()

	dbRec := &route53model.Route53Domain{}
	res := db.Where(&route53model.Route53Domain{OrganizationId: orgId, Domain: domain}).First(dbRec)

	if res.RecordNotFound() {
		return false, nil
	}
	err := res.Error
	if err != nil {
		return false, nil
	}

	initStateFromRoute53Domain(dbRec, stateOut)

	return true, nil
}

// listUnused returns all the domain state entries from database that belong to organizations with no live clusters
// thus the DNS domain entries earlier created for these domain are not used any more
func (stateStore *awsRoute53DatabaseStateStore) listUnused() ([]domainState, error) {
	db := model.GetDB()
	var dbRecs []route53model.Route53Domain

	sqlFilter := fmt.Sprintf("organization_id NOT IN (SELECT organization_id FROM clusters WHERE deleted_at is NULL AND status<>'%s')", cluster.Error)

	err := db.Where(&route53model.Route53Domain{Status: CREATED}).Where(sqlFilter).Find(&dbRecs).Error
	if err != nil {
		return nil, err
	}

	var domainStates []domainState

	for i := 0; i < len(dbRecs); i++ {
		var state domainState

		initStateFromRoute53Domain(&dbRecs[i], &state)
		domainStates = append(domainStates, state)
	}

	return domainStates, nil
}

// delete deletes domain state from database
func (stateStore *awsRoute53DatabaseStateStore) delete(state *domainState) error {
	db := model.GetDB()

	crit := &route53model.Route53Domain{OrganizationId: state.organisationId, Domain: state.domain}

	return db.Where(crit).Delete(&route53model.Route53Domain{}).Error
}

// createRoute53Domain create a new Route53Domain instance initialized from the passed in state
func createRoute53Domain(state *domainState) *route53model.Route53Domain {
	return &route53model.Route53Domain{
		OrganizationId: state.organisationId,
		Domain:         state.domain,
		Status:         state.status,
		PolicyArn:      state.policyArn,
		HostedZoneId:   state.hostedZoneId,
		IamUser:        state.iamUser,
		AwsAccessKeyId: state.awsAccessKeyId,
		ErrorMessage:   state.errMsg,
	}
}

// initStateFromRoute53Domain initializes state from the passed db record
func initStateFromRoute53Domain(dbRecord *route53model.Route53Domain, state *domainState) {
	state.createdAt = dbRecord.CreatedAt
	state.organisationId = dbRecord.OrganizationId
	state.domain = dbRecord.Domain
	state.status = dbRecord.Status
	state.policyArn = dbRecord.PolicyArn
	state.hostedZoneId = dbRecord.HostedZoneId
	state.iamUser = dbRecord.IamUser
	state.awsAccessKeyId = dbRecord.AwsAccessKeyId
	state.errMsg = dbRecord.ErrorMessage
}
