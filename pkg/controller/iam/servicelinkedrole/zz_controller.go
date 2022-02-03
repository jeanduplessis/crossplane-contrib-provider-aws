/*
Copyright 2021 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by ack-generate. DO NOT EDIT.

package servicelinkedrole

import (
	"context"

	svcapi "github.com/aws/aws-sdk-go/service/iam"
	svcsdk "github.com/aws/aws-sdk-go/service/iam"
	svcsdkapi "github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	cpresource "github.com/crossplane/crossplane-runtime/pkg/resource"

	svcapitypes "github.com/crossplane-contrib/provider-aws/apis/iam/v1alpha1"
	awsclient "github.com/crossplane-contrib/provider-aws/pkg/clients"
)

const (
	errUnexpectedObject = "managed resource is not an ServiceLinkedRole resource"

	errCreateSession = "cannot create a new session"
	errCreate        = "cannot create ServiceLinkedRole in AWS"
	errUpdate        = "cannot update ServiceLinkedRole in AWS"
	errDescribe      = "failed to describe ServiceLinkedRole"
	errDelete        = "failed to delete ServiceLinkedRole"
)

type connector struct {
	kube client.Client
	opts []option
}

func (c *connector) Connect(ctx context.Context, mg cpresource.Managed) (managed.ExternalClient, error) {
	sess, err := awsclient.GetConfigV1(ctx, c.kube, mg, awsclient.GlobalRegion)
	if err != nil {
		return nil, errors.Wrap(err, errCreateSession)
	}
	return newExternal(c.kube, svcapi.New(sess), c.opts), nil
}

func (e *external) Observe(ctx context.Context, mg cpresource.Managed) (managed.ExternalObservation, error) {
	return e.observe(ctx, mg)
}

func (e *external) Create(ctx context.Context, mg cpresource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*svcapitypes.ServiceLinkedRole)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}
	cr.Status.SetConditions(xpv1.Creating())
	input := GenerateCreateServiceLinkedRoleInput(cr)
	if err := e.preCreate(ctx, cr, input); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "pre-create failed")
	}
	resp, err := e.client.CreateServiceLinkedRoleWithContext(ctx, input)
	if err != nil {
		return managed.ExternalCreation{}, awsclient.Wrap(err, errCreate)
	}

	if resp.Role.Arn != nil {
		cr.Status.AtProvider.ARN = resp.Role.Arn
	} else {
		cr.Status.AtProvider.ARN = nil
	}
	if resp.Role.AssumeRolePolicyDocument != nil {
		cr.Status.AtProvider.AssumeRolePolicyDocument = resp.Role.AssumeRolePolicyDocument
	} else {
		cr.Status.AtProvider.AssumeRolePolicyDocument = nil
	}
	if resp.Role.CreateDate != nil {
		cr.Status.AtProvider.CreateDate = &metav1.Time{*resp.Role.CreateDate}
	} else {
		cr.Status.AtProvider.CreateDate = nil
	}
	if resp.Role.Description != nil {
		cr.Spec.ForProvider.Description = resp.Role.Description
	} else {
		cr.Spec.ForProvider.Description = nil
	}
	if resp.Role.MaxSessionDuration != nil {
		cr.Status.AtProvider.MaxSessionDuration = resp.Role.MaxSessionDuration
	} else {
		cr.Status.AtProvider.MaxSessionDuration = nil
	}
	if resp.Role.Path != nil {
		cr.Status.AtProvider.Path = resp.Role.Path
	} else {
		cr.Status.AtProvider.Path = nil
	}
	if resp.Role.PermissionsBoundary != nil {
		f6 := &svcapitypes.AttachedPermissionsBoundary{}
		if resp.Role.PermissionsBoundary.PermissionsBoundaryArn != nil {
			f6.PermissionsBoundaryARN = resp.Role.PermissionsBoundary.PermissionsBoundaryArn
		}
		if resp.Role.PermissionsBoundary.PermissionsBoundaryType != nil {
			f6.PermissionsBoundaryType = resp.Role.PermissionsBoundary.PermissionsBoundaryType
		}
		cr.Status.AtProvider.PermissionsBoundary = f6
	} else {
		cr.Status.AtProvider.PermissionsBoundary = nil
	}
	if resp.Role.RoleId != nil {
		cr.Status.AtProvider.RoleID = resp.Role.RoleId
	} else {
		cr.Status.AtProvider.RoleID = nil
	}
	if resp.Role.RoleLastUsed != nil {
		f8 := &svcapitypes.RoleLastUsed{}
		if resp.Role.RoleLastUsed.LastUsedDate != nil {
			f8.LastUsedDate = &metav1.Time{*resp.Role.RoleLastUsed.LastUsedDate}
		}
		if resp.Role.RoleLastUsed.Region != nil {
			f8.Region = resp.Role.RoleLastUsed.Region
		}
		cr.Status.AtProvider.RoleLastUsed = f8
	} else {
		cr.Status.AtProvider.RoleLastUsed = nil
	}
	if resp.Role.RoleName != nil {
		cr.Status.AtProvider.RoleName = resp.Role.RoleName
	} else {
		cr.Status.AtProvider.RoleName = nil
	}
	if resp.Role.Tags != nil {
		f10 := []*svcapitypes.Tag{}
		for _, f10iter := range resp.Role.Tags {
			f10elem := &svcapitypes.Tag{}
			if f10iter.Key != nil {
				f10elem.Key = f10iter.Key
			}
			if f10iter.Value != nil {
				f10elem.Value = f10iter.Value
			}
			f10 = append(f10, f10elem)
		}
		cr.Status.AtProvider.Tags = f10
	} else {
		cr.Status.AtProvider.Tags = nil
	}

	return e.postCreate(ctx, cr, resp, managed.ExternalCreation{}, err)
}

func (e *external) Update(ctx context.Context, mg cpresource.Managed) (managed.ExternalUpdate, error) {
	return e.update(ctx, mg)

}

func (e *external) Delete(ctx context.Context, mg cpresource.Managed) error {
	cr, ok := mg.(*svcapitypes.ServiceLinkedRole)
	if !ok {
		return errors.New(errUnexpectedObject)
	}
	cr.Status.SetConditions(xpv1.Deleting())
	input := GenerateDeleteServiceLinkedRoleInput(cr)
	ignore, err := e.preDelete(ctx, cr, input)
	if err != nil {
		return errors.Wrap(err, "pre-delete failed")
	}
	if ignore {
		return nil
	}
	resp, err := e.client.DeleteServiceLinkedRoleWithContext(ctx, input)
	return e.postDelete(ctx, cr, resp, awsclient.Wrap(cpresource.Ignore(IsNotFound, err), errDelete))
}

type option func(*external)

func newExternal(kube client.Client, client svcsdkapi.IAMAPI, opts []option) *external {
	e := &external{
		kube:       kube,
		client:     client,
		observe:    nopObserve,
		preCreate:  nopPreCreate,
		postCreate: nopPostCreate,
		preDelete:  nopPreDelete,
		postDelete: nopPostDelete,
		update:     nopUpdate,
	}
	for _, f := range opts {
		f(e)
	}
	return e
}

type external struct {
	kube       client.Client
	client     svcsdkapi.IAMAPI
	observe    func(context.Context, cpresource.Managed) (managed.ExternalObservation, error)
	preCreate  func(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.CreateServiceLinkedRoleInput) error
	postCreate func(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.CreateServiceLinkedRoleOutput, managed.ExternalCreation, error) (managed.ExternalCreation, error)
	preDelete  func(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.DeleteServiceLinkedRoleInput) (bool, error)
	postDelete func(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.DeleteServiceLinkedRoleOutput, error) error
	update     func(context.Context, cpresource.Managed) (managed.ExternalUpdate, error)
}

func nopObserve(context.Context, cpresource.Managed) (managed.ExternalObservation, error) {
	return managed.ExternalObservation{}, nil
}

func nopPreCreate(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.CreateServiceLinkedRoleInput) error {
	return nil
}
func nopPostCreate(_ context.Context, _ *svcapitypes.ServiceLinkedRole, _ *svcsdk.CreateServiceLinkedRoleOutput, cre managed.ExternalCreation, err error) (managed.ExternalCreation, error) {
	return cre, err
}
func nopPreDelete(context.Context, *svcapitypes.ServiceLinkedRole, *svcsdk.DeleteServiceLinkedRoleInput) (bool, error) {
	return false, nil
}
func nopPostDelete(_ context.Context, _ *svcapitypes.ServiceLinkedRole, _ *svcsdk.DeleteServiceLinkedRoleOutput, err error) error {
	return err
}
func nopUpdate(context.Context, cpresource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}
