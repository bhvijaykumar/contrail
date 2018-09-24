package logic

import (
	"context"

	"github.com/pkg/errors"

	"github.com/Juniper/contrail/pkg/common"
	"github.com/Juniper/contrail/pkg/compilation/intent"
	"github.com/Juniper/contrail/pkg/models"
	"github.com/Juniper/contrail/pkg/models/basemodels"
	"github.com/Juniper/contrail/pkg/services"
)

// SecurityGroupIntent contains Intent Compiler state for SecurityGroup
type SecurityGroupIntent struct {
	intent.BaseIntent
	*models.SecurityGroup

	ingressACL, egressACL *models.AccessControlList
}

// GetObject returns embedded resource object
func (i *SecurityGroupIntent) GetObject() basemodels.Object {
	return i.SecurityGroup
}

// CreateSecurityGroup evaluates SecurityGroup dependencies.
func (s *Service) CreateSecurityGroup(
	ctx context.Context,
	request *services.CreateSecurityGroupRequest,
) (*services.CreateSecurityGroupResponse, error) {
	i := &SecurityGroupIntent{
		SecurityGroup: request.GetSecurityGroup(),
	}

	if err := s.handleCreate(ctx, i); err != nil {
		return nil, err
	}

	return s.BaseService.CreateSecurityGroup(ctx, request)
}

// DeleteSecurityGroup evaluates SecurityGroup dependencies.
func (s *Service) DeleteSecurityGroup(
	ctx context.Context,
	request *services.DeleteSecurityGroupRequest,
) (*services.DeleteSecurityGroupResponse, error) {

	i := loadSecurityGroupIntent(s.cache, intent.ByUUID(request.GetID()))
	if i == nil {
		return nil, errors.New("failed to process SecurityGroup deletion: SecurityGroupIntent not found in cache")
	}

	if err := deleteDefaultACLs(ctx, s.WriteService, i); err != nil {
		return nil, errors.Wrap(err, "failed to process SecurityGroup deletion")
	}

	s.cache.Delete(models.KindSecurityGroup, intent.ByUUID(i.GetUUID()))

	return s.BaseService.DeleteSecurityGroup(ctx, request)
}

func deleteDefaultACLs(ctx context.Context, writeService services.WriteService, i *SecurityGroupIntent) error {
	var multiError common.MultiError

	if err := deleteACLIfNeeded(ctx, writeService, i.ingressACL); err != nil {
		multiError = append(multiError, errors.Wrap(err, "failed to delete ingress access control list"))
	} else {
		i.ingressACL = nil
	}

	if err := deleteACLIfNeeded(ctx, writeService, i.egressACL); err != nil {
		multiError = append(multiError, errors.Wrap(err, "failed to delete egress access control list"))
	} else {
		i.egressACL = nil
	}

	if len(multiError) > 0 {
		return multiError
	}
	return nil
}

func deleteACLIfNeeded(
	ctx context.Context, writeService services.WriteService, acl *models.AccessControlList,
) error {
	if acl == nil {
		return nil
	}
	return deleteACL(ctx, writeService, acl.GetUUID())
}

// Evaluate Creates default AccessControlList's for the already created SecurityGroup.
func (i *SecurityGroupIntent) Evaluate(ctx context.Context, ec *intent.EvaluateContext) error {
	ingressACL, egressACL := i.DefaultACLs(ec)

	// TODO: Use batch create so that either both ACLs are created or none.
	var err error
	i.ingressACL, err = createACL(ctx, ec.WriteService, ingressACL)
	if err != nil {
		return errors.Wrap(err, "failed to create ingress access control list")
	}

	i.egressACL, err = createACL(ctx, ec.WriteService, egressACL)
	if err != nil {
		return errors.Wrap(err, "failed to create egress access control list")
	}

	return nil
}

// DefaultACLs returns default ACLs corresponding to the security group's policy rules.
func (i *SecurityGroupIntent) DefaultACLs(ec *intent.EvaluateContext) (
	ingressACL *models.AccessControlList, egressACL *models.AccessControlList) {

	rs := &models.PolicyRulesWithRefs{
		Rules:      i.GetSecurityGroupEntries().GetPolicyRule(),
		FQNameToSG: make(map[string]*models.SecurityGroup),
	}
	resolveSGRefs(rs, ec)

	ingressRules, egressRules := rs.ToACLRules()

	ingressACL = i.MakeChildACL("ingress-access-control-list", ingressRules)
	egressACL = i.MakeChildACL("egress-access-control-list", egressRules)
	return ingressACL, egressACL
}

func resolveSGRefs(rs *models.PolicyRulesWithRefs, ec *intent.EvaluateContext) {
	for _, r := range rs.Rules {
		for _, addr := range r.SRCAddresses {
			resolveSGRef(rs, addr, ec)
		}
		for _, addr := range r.DSTAddresses {
			resolveSGRef(rs, addr, ec)
		}
	}
}

func resolveSGRef(rs *models.PolicyRulesWithRefs, addr *models.AddressType, ec *intent.EvaluateContext) {
	if !addr.IsSecurityGroupNameAReference() {
		return
	}
	i := loadSecurityGroupIntent(
		ec.IntentLoader,
		intent.ByFQName(basemodels.ParseFQName(addr.SecurityGroup)))
	rs.FQNameToSG[addr.SecurityGroup] = i.SecurityGroup
}

func loadSecurityGroupIntent(loader intent.Loader, query intent.Query) *SecurityGroupIntent {
	intent := loader.Load(models.KindSecurityGroup, query)
	sgIntent, _ := intent.(*SecurityGroupIntent)
	return sgIntent
}

func createACL(
	ctx context.Context, writeService services.WriteService, acl *models.AccessControlList,
) (*models.AccessControlList, error) {
	response, err := writeService.CreateAccessControlList(
		ctx, &services.CreateAccessControlListRequest{
			AccessControlList: acl,
		})
	return response.GetAccessControlList(), err
}

func deleteACL(ctx context.Context, writeService services.WriteService, uuid string) error {
	_, err := writeService.DeleteAccessControlList(
		ctx, &services.DeleteAccessControlListRequest{
			ID: uuid,
		})
	return err
}
