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

package configurationset

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/aws/aws-sdk-go/service/sesv2/sesv2iface"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	svcapitypes "github.com/crossplane-contrib/provider-aws/apis/sesv2/v1alpha1"
	awsclient "github.com/crossplane-contrib/provider-aws/pkg/clients"
	svcutils "github.com/crossplane-contrib/provider-aws/pkg/controller/sesv2"

	"github.com/pkg/errors"
)

const (
	errNotConfigurationSet = "managed resource is not a SES ConfigurationSet custom resource"
	errKubeUpdateFailed    = "cannot update SES ConfigurationSet custom resource"
)

// SetupConfigurationSet adds a controller that reconciles SES ConfigurationSet.
func SetupConfigurationSet(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(svcapitypes.ConfigurationSetGroupKind)
	opts := []option{
		func(e *external) {
			h := &hooks{client: e.client, kube: e.kube}
			e.isUpToDate = isUpToDate
			e.preObserve = preObserve
			e.postObserve = h.postObserve
			e.preCreate = preCreate
			e.preDelete = preDelete
			e.update = h.update
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&svcapitypes.ConfigurationSet{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(svcapitypes.ConfigurationSetGroupVersionKind),
			managed.WithExternalConnecter(&connector{kube: mgr.GetClient(), opts: opts}),
			managed.WithInitializers(managed.NewDefaultProviderConfig(mgr.GetClient()), managed.NewNameAsExternalName(mgr.GetClient()), &tagger{kube: mgr.GetClient()}),
			managed.WithPollInterval(o.PollInterval),
			managed.WithLogger(o.Logger.WithValues("controller", name)),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

type hooks struct {
	client                      sesv2iface.SESV2API
	kube                        client.Client
	ConfigurationSetObservation *svcsdk.GetConfigurationSetOutput
}

func isUpToDate(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) (bool, error) {
	if meta.WasDeleted(cr) {
		return true, nil // There is no need to check for updates when we want to delete.
	}

	if !isUpToDateDeliveryOptions(cr, resp) {
		return false, nil
	}

	if !isUpToDateReputationOptions(cr, resp) {
		return false, nil
	}

	if !isUpToDateSendingOptions(cr, resp) {
		return false, nil
	}

	if !isUpToDateSuppressionOptions(cr, resp) {
		return false, nil
	}

	if !isUpToDateTrackingOptions(cr, resp) {
		return false, nil
	}

	return svcutils.AreTagsUpToDate(cr.Spec.ForProvider.Tags, resp.Tags)
}

// isUpToDateDeliveryOptions checks if DeliveryOptions Object are up to date
func isUpToDateDeliveryOptions(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) bool {
	if cr.Spec.ForProvider.DeliveryOptions != nil && resp.DeliveryOptions != nil {
		if awsclient.StringValue(cr.Spec.ForProvider.DeliveryOptions.SendingPoolName) != awsclient.StringValue(resp.DeliveryOptions.SendingPoolName) {
			return false
		}
		if awsclient.StringValue(cr.Spec.ForProvider.DeliveryOptions.TLSPolicy) != awsclient.StringValue(resp.DeliveryOptions.TlsPolicy) {
			return false
		}
	}
	return true
}

// isUpToDateReputationOptions checks if ReputationOptions Object are up to date
func isUpToDateReputationOptions(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) bool {
	if cr.Spec.ForProvider.ReputationOptions != nil && resp.ReputationOptions != nil {
		if awsclient.BoolValue(cr.Spec.ForProvider.ReputationOptions.ReputationMetricsEnabled) != awsclient.BoolValue(resp.ReputationOptions.ReputationMetricsEnabled) {
			return false
		}
	}
	return true
}

// isUpToDateTrackingOptions checks if TrackingOptions Object are up to date
func isUpToDateTrackingOptions(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) bool {
	// Once disabled, output response will not populate this option anymore
	if cr.Spec.ForProvider.TrackingOptions != nil && resp.TrackingOptions == nil {
		return false
	}

	if cr.Spec.ForProvider.TrackingOptions != nil && resp.TrackingOptions != nil {
		if awsclient.StringValue(cr.Spec.ForProvider.TrackingOptions.CustomRedirectDomain) != awsclient.StringValue(resp.TrackingOptions.CustomRedirectDomain) {
			return false
		}
	}
	return true
}

// isUpToDateSuppressionOptions checks if SuppressionOptions Object are up to date
func isUpToDateSuppressionOptions(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) bool {
	var crSuppressedReasons []*string
	var awsSuppressedReasons []*string

	if cr.Spec.ForProvider.SuppressionOptions != nil && cr.Spec.ForProvider.SuppressionOptions.SuppressedReasons != nil {
		crSuppressedReasons = cr.Spec.ForProvider.SuppressionOptions.SuppressedReasons
	}

	// SuppressedReasons Response return empty slice if not being configured (e.g. "SuppressedReasons": [])
	if resp.SuppressionOptions != nil && resp.SuppressionOptions.SuppressedReasons != nil {
		awsSuppressedReasons = resp.SuppressionOptions.SuppressedReasons
	}

	if len(crSuppressedReasons) != len(awsSuppressedReasons) {
		return false
	}

	sortCmp := cmpopts.SortSlices(func(i, j *string) bool {
		return aws.StringValue(i) < aws.StringValue(j)
	})

	return cmp.Equal(crSuppressedReasons, awsSuppressedReasons, sortCmp, cmpopts.EquateEmpty())

}

// isUpToDateSendingOptions checks if SendingOptions Object are up to date
func isUpToDateSendingOptions(cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput) bool {
	if cr.Spec.ForProvider.SendingOptions != nil && resp.SendingOptions != nil {
		if awsclient.BoolValue(cr.Spec.ForProvider.SendingOptions.SendingEnabled) != awsclient.BoolValue(resp.SendingOptions.SendingEnabled) {
			return false
		}
	}
	return true
}

func preObserve(_ context.Context, cr *svcapitypes.ConfigurationSet, obj *svcsdk.GetConfigurationSetInput) error {
	obj.ConfigurationSetName = awsclient.String(meta.GetExternalName(cr))
	return nil
}

func (e *hooks) postObserve(_ context.Context, cr *svcapitypes.ConfigurationSet, resp *svcsdk.GetConfigurationSetOutput, obs managed.ExternalObservation, err error) (managed.ExternalObservation, error) {
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	switch awsclient.BoolValue(resp.SendingOptions.SendingEnabled) {
	case true:
		cr.Status.SetConditions(xpv1.Available())
	case false:
		cr.Status.SetConditions(xpv1.Unavailable())
	default:
		cr.Status.SetConditions(xpv1.Creating())
	}

	// Passing ConfigurationSet object from Observation into hooks for Update function to access
	e.ConfigurationSetObservation = resp

	return obs, nil
}

func preCreate(_ context.Context, cr *svcapitypes.ConfigurationSet, obj *svcsdk.CreateConfigurationSetInput) error {
	obj.ConfigurationSetName = awsclient.String(meta.GetExternalName(cr))
	return nil
}

func preDelete(_ context.Context, cr *svcapitypes.ConfigurationSet, obj *svcsdk.DeleteConfigurationSetInput) (bool, error) {
	obj.ConfigurationSetName = awsclient.String(meta.GetExternalName(cr))
	return false, nil
}

func (e *hooks) update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) { // nolint:gocyclo
	cr, ok := mg.(*svcapitypes.ConfigurationSet)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}
	// Update Resource is not provided other than individual PUT operation
	// NOTE: Update operation NOT generated by ACK code-generator

	// Populate ConfigurationSetName from meta.AnnotationKeyExternalName
	configurationSetName := awsclient.String(mg.GetAnnotations()[meta.AnnotationKeyExternalName])

	// update DeliveryOptions (PutConfigurationSetDeliveryOptions)
	if !isUpToDateDeliveryOptions(cr, e.ConfigurationSetObservation) {
		deliveryOptionsInput := &svcsdk.PutConfigurationSetDeliveryOptionsInput{
			ConfigurationSetName: configurationSetName,
			SendingPoolName:      cr.Spec.ForProvider.DeliveryOptions.SendingPoolName,
			TlsPolicy:            cr.Spec.ForProvider.DeliveryOptions.TLSPolicy,
		}
		if _, err := e.client.PutConfigurationSetDeliveryOptionsWithContext(ctx, deliveryOptionsInput); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "update failed for ConfigurationSetDeliveryOptions")
		}
	}

	// update ReputationOptions (PutConfigurationSetReputationOptions)
	if !isUpToDateReputationOptions(cr, e.ConfigurationSetObservation) {
		reputationOptionsInput := &svcsdk.PutConfigurationSetReputationOptionsInput{
			ConfigurationSetName:     configurationSetName,
			ReputationMetricsEnabled: cr.Spec.ForProvider.ReputationOptions.ReputationMetricsEnabled,
		}
		if _, err := e.client.PutConfigurationSetReputationOptionsWithContext(ctx, reputationOptionsInput); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "update failed for ConfigurationSetReputationOptions")
		}
	}

	// update SuppressionOptions (PutConfigurationSetSuppressionOptions)
	var suppresssedReasons []*string
	if !isUpToDateSuppressionOptions(cr, e.ConfigurationSetObservation) {
		suppresssedReasons = cr.Spec.ForProvider.SuppressionOptions.SuppressedReasons
		supressOptionsInput := &svcsdk.PutConfigurationSetSuppressionOptionsInput{
			ConfigurationSetName: configurationSetName,
			SuppressedReasons:    suppresssedReasons,
		}
		if _, err := e.client.PutConfigurationSetSuppressionOptionsWithContext(ctx, supressOptionsInput); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "update failed for ConfigurationSetSuppressionOptions")
		}
	}

	// update TrackingOptions (PutConfigurationSetTrackingOptions)
	if !isUpToDateTrackingOptions(cr, e.ConfigurationSetObservation) {
		trackingOptionInput := &svcsdk.PutConfigurationSetTrackingOptionsInput{
			ConfigurationSetName: configurationSetName,
			CustomRedirectDomain: cr.Spec.ForProvider.TrackingOptions.CustomRedirectDomain,
		}
		if _, err := e.client.PutConfigurationSetTrackingOptionsWithContext(ctx, trackingOptionInput); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "update failed for ConfigurationSetTrackingOptions")
		}
	}

	// update SendingOptions (PutConfigurationSetSendingOptions)
	if !isUpToDateSendingOptions(cr, e.ConfigurationSetObservation) {
		sendingOptionInput := &svcsdk.PutConfigurationSetSendingOptionsInput{
			ConfigurationSetName: configurationSetName,
			SendingEnabled:       cr.Spec.ForProvider.SendingOptions.SendingEnabled,
		}
		if _, err := e.client.PutConfigurationSetSendingOptionsWithContext(ctx, sendingOptionInput); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "update failed for ConfigurationSetSendingOptions")
		}
	}

	return managed.ExternalUpdate{}, nil
}

type tagger struct {
	kube client.Client
}

func (t *tagger) Initialize(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*svcapitypes.ConfigurationSet)
	if !ok {
		return errors.New(errNotConfigurationSet)
	}
	cr.Spec.ForProvider.Tags = svcutils.AddExternalTags(mg, cr.Spec.ForProvider.Tags)
	return errors.Wrap(t.kube.Update(ctx, cr), errKubeUpdateFailed)
}
