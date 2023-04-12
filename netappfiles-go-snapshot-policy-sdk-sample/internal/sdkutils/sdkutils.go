// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// This package centralizes any function that directly
// using any of the Azure's (with exception of authentication related ones)
// available SDK packages.

package sdkutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/iam"
	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/uri"
	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/utils"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/netapp/armnetapp"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

const (
	userAgent = "anf-go-sdk-sample-agent"
	nfsv3     = "NFSv3"
	nfsv41    = "NFSv4.1"
	cifs      = "CIFS"
)

var (
	validProtocols = []string{nfsv3, nfsv41, cifs}
)

func validateANFServiceLevel(serviceLevel string) (validatedServiceLevel armnetapp.ServiceLevel, err error) {
	var svcLevel armnetapp.ServiceLevel

	switch strings.ToLower(serviceLevel) {
	case "ultra":
		svcLevel = armnetapp.ServiceLevelUltra
	case "premium":
		svcLevel = armnetapp.ServiceLevelPremium
	case "standard":
		svcLevel = armnetapp.ServiceLevelStandard
	default:
		return "", fmt.Errorf("invalid service level, supported service levels are: %v", armnetapp.PossibleServiceLevelValues())
	}

	return svcLevel, nil
}

func getResourcesClient() (*armresources.Client, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armresources.NewClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getAccountsClient() (*armnetapp.AccountsClient, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armnetapp.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getPoolsClient() (*armnetapp.PoolsClient, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armnetapp.NewPoolsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getVolumesClient() (*armnetapp.VolumesClient, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armnetapp.NewVolumesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getSnapshotsClient() (*armnetapp.SnapshotsClient, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armnetapp.NewSnapshotsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getSnapshotPoliciesClient() (*armnetapp.SnapshotPoliciesClient, error) {
	cred, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return nil, err
	}

	client, err := armnetapp.NewSnapshotPoliciesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetResourceByID gets a generic resource
func GetResourceByID(ctx context.Context, resourceID, APIVersion string) (armresources.ClientGetResponse, error) {
	resourcesClient, err := getResourcesClient()
	if err != nil {
		return armresources.ClientGetResponse{}, err
	}

	parentResource := ""
	resourceGroup := uri.GetResourceGroup(resourceID)
	resourceProvider := uri.GetResourceValue(resourceID, "providers")
	resourceName := uri.GetResourceName(resourceID)
	resourceType := uri.GetResourceValue(resourceID, resourceProvider)

	if strings.Contains(resourceID, "/subnets/") {
		parentResourceName := uri.GetResourceValue(resourceID, resourceType)
		parentResource = fmt.Sprintf("%v/%v", resourceType, parentResourceName)
		resourceType = "subnets"
	}

	return resourcesClient.Get(
		ctx,
		resourceGroup,
		resourceProvider,
		parentResource,
		resourceType,
		resourceName,
		APIVersion,
		nil,
	)
}

// CreateANFAccount creates an ANF Account resource
func CreateANFAccount(ctx context.Context, location, resourceGroupName, accountName string, activeDirectories []*armnetapp.ActiveDirectory, tags map[string]*string) (*armnetapp.Account, error) {
	accountClient, err := getAccountsClient()
	if err != nil {
		return nil, err
	}

	accountProperties := armnetapp.AccountProperties{}

	if activeDirectories != nil {
		accountProperties = armnetapp.AccountProperties{
			ActiveDirectories: activeDirectories,
		}
	}

	future, err := accountClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		accountName,
		armnetapp.Account{
			Location:   to.Ptr(location),
			Tags:       tags,
			Properties: &accountProperties,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create account: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get the account create or update future response: %v", err)
	}

	return &resp.Account, nil
}

// CreateANFCapacityPool creates an ANF Capacity Pool within ANF Account
func CreateANFCapacityPool(ctx context.Context, location, resourceGroupName, accountName, poolName, serviceLevel string, sizeBytes int64, tags map[string]*string) (*armnetapp.CapacityPool, error) {
	poolClient, err := getPoolsClient()
	if err != nil {
		return nil, err
	}

	svcLevel, err := validateANFServiceLevel(serviceLevel)
	if err != nil {
		return nil, err
	}

	future, err := poolClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		armnetapp.CapacityPool{
			Location: to.Ptr(location),
			Tags:     tags,
			Properties: &armnetapp.PoolProperties{
				ServiceLevel: &svcLevel,
				Size:         to.Ptr[int64](sizeBytes),
			},
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create pool: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get the pool create or update future response: %v", err)
	}

	return &resp.CapacityPool, nil
}

// CreateANFVolume creates an ANF volume within a Capacity Pool
func CreateANFVolume(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName, serviceLevel, subnetID, snapshotID string, protocolTypes []string, volumeUsageQuota int64, unixReadOnly, unixReadWrite bool, tags map[string]*string, dataProtectionObject armnetapp.VolumePropertiesDataProtection) (*armnetapp.Volume, error) {
	if len(protocolTypes) > 2 {
		return nil, fmt.Errorf("maximum of two protocol types are supported")
	}

	if len(protocolTypes) > 1 && utils.Contains(protocolTypes, "NFSv4.1") {
		return nil, fmt.Errorf("only cifs/nfsv3 protocol types are supported as dual protocol")
	}

	_, found := utils.FindInSlice(validProtocols, protocolTypes[0])
	if !found {
		return nil, fmt.Errorf("invalid protocol type, valid protocol types are: %v", validProtocols)
	}

	svcLevel, err := validateANFServiceLevel(serviceLevel)
	if err != nil {
		return nil, err
	}

	volumeClient, err := getVolumesClient()
	if err != nil {
		return nil, err
	}

	exportPolicy := armnetapp.VolumePropertiesExportPolicy{}

	if _, found := utils.FindInSlice(protocolTypes, cifs); !found {
		exportPolicy = armnetapp.VolumePropertiesExportPolicy{
			Rules: []*armnetapp.ExportPolicyRule{
				{
					AllowedClients: to.Ptr("0.0.0.0/0"),
					Cifs:           to.Ptr(map[bool]bool{true: true, false: false}[protocolTypes[0] == cifs]),
					Nfsv3:          to.Ptr(map[bool]bool{true: true, false: false}[protocolTypes[0] == nfsv3]),
					Nfsv41:         to.Ptr(map[bool]bool{true: true, false: false}[protocolTypes[0] == nfsv41]),
					RuleIndex:      to.Ptr[int32](1),
					UnixReadOnly:   to.Ptr(unixReadOnly),
					UnixReadWrite:  to.Ptr(unixReadWrite),
				},
			},
		}
	}

	protocolTypeSlice := make([]*string, len(protocolTypes))
	for i, protocolType := range protocolTypes {
		protocolTypeSlice[i] = &protocolType
	}

	volumeProperties := armnetapp.VolumeProperties{
		SnapshotID:     map[bool]*string{true: to.Ptr(snapshotID), false: nil}[snapshotID != ""],
		ExportPolicy:   map[bool]*armnetapp.VolumePropertiesExportPolicy{true: &exportPolicy, false: nil}[protocolTypes[0] != cifs],
		ProtocolTypes:  protocolTypeSlice,
		ServiceLevel:   &svcLevel,
		SubnetID:       to.Ptr(subnetID),
		UsageThreshold: to.Ptr[int64](volumeUsageQuota),
		CreationToken:  to.Ptr(volumeName),
		DataProtection: &dataProtectionObject,
	}

	future, err := volumeClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		armnetapp.Volume{
			Location:   to.Ptr(location),
			Tags:       tags,
			Properties: &volumeProperties,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create volume: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get the volume create or update future response: %v", err)
	}

	return &resp.Volume, nil
}

// UpdateANFVolume update an ANF volume
func UpdateANFVolume(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName string, volumePropertiesPatch armnetapp.VolumePatchProperties, tags map[string]*string) (*armnetapp.Volume, error) {
	volumeClient, err := getVolumesClient()
	if err != nil {
		return nil, err
	}

	future, err := volumeClient.BeginUpdate(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		armnetapp.VolumePatch{
			Location:   to.Ptr(location),
			Tags:       tags,
			Properties: &volumePropertiesPatch,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot update volume: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.Volume, nil
}

// AuthorizeReplication - authorizes volume replication
func AuthorizeReplication(ctx context.Context, resourceGroupName, accountName, poolName, volumeName, remoteVolumeResourceID string) error {
	volumeClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumeClient.BeginAuthorizeReplication(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		armnetapp.AuthorizeRequest{
			RemoteVolumeResourceID: to.Ptr(remoteVolumeResourceID),
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot authorize volume replication: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get authorize volume replication future response: %v", err)
	}

	return nil
}

// DeleteANFVolumeReplication - authorizes volume replication
func DeleteANFVolumeReplication(ctx context.Context, resourceGroupName, accountName, poolName, volumeName string) error {
	volumeClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumeClient.BeginDeleteReplication(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete volume replication: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get delete volume replication future response: %v", err)
	}

	return nil
}

// CreateANFSnapshot creates a Snapshot from an ANF volume
func CreateANFSnapshot(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName, snapshotName string, tags map[string]*string) (*armnetapp.Snapshot, error) {

	snapshotClient, err := getSnapshotsClient()
	if err != nil {
		return nil, err
	}

	future, err := snapshotClient.BeginCreate(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		snapshotName,
		armnetapp.Snapshot{
			Location: to.Ptr(location),
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create snapshot: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get the snapshot create or update future response: %v", err)
	}

	return &resp.Snapshot, nil
}

// DeleteANFSnapshot deletes a Snapshot from an ANF volume
func DeleteANFSnapshot(ctx context.Context, resourceGroupName, accountName, poolName, volumeName, snapshotName string) error {
	snapshotClient, err := getSnapshotsClient()
	if err != nil {
		return err
	}

	future, err := snapshotClient.BeginDelete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		snapshotName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete snapshot: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get the snapshot delete future response: %v", err)
	}

	return nil
}

// CreateANFSnapshotPolicy creates a Snapshot Policy to be used on volumes
func CreateANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string, policy armnetapp.SnapshotPolicy) (*armnetapp.SnapshotPolicy, error) {
	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return nil, err
	}

	snapshotPolicy, err := snapshotPolicyClient.Create(
		ctx,
		resourceGroupName,
		accountName,
		policyName,
		policy,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create snapshot policy: %v", err)
	}

	return &snapshotPolicy.SnapshotPolicy, nil
}

// UpdateANFSnapshotPolicy update an ANF volume
func UpdateANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string, snapshotPolicyPatch armnetapp.SnapshotPolicyPatch) (*armnetapp.SnapshotPolicy, error) {
	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return nil, err
	}

	future, err := snapshotPolicyClient.BeginUpdate(
		ctx,
		resourceGroupName,
		accountName,
		policyName,
		snapshotPolicyPatch,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot update snapshot policy: %v", err)
	}

	resp, err := future.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &resp.SnapshotPolicy, nil
}

// DeleteANFVolume deletes a volume
func DeleteANFVolume(ctx context.Context, resourceGroupName, accountName, poolName, volumeName string) error {
	volumesClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumesClient.BeginDelete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete volume: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get the volume delete future response: %v", err)
	}

	return nil
}

// DeleteANFCapacityPool deletes a capacity pool
func DeleteANFCapacityPool(ctx context.Context, resourceGroupName, accountName, poolName string) error {
	poolsClient, err := getPoolsClient()
	if err != nil {
		return err
	}

	future, err := poolsClient.BeginDelete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete capacity pool: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get the capacity pool delete future response: %v", err)
	}

	return nil
}

// DeleteANFSnapshotPolicy deletes a snapshot policy
func DeleteANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string) error {
	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return err
	}

	future, err := snapshotPolicyClient.BeginDelete(
		ctx,
		resourceGroupName,
		accountName,
		policyName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete snapshot policy: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get the snapshot policy delete future response: %v", err)
	}

	return nil
}

// DeleteANFAccount deletes an account
func DeleteANFAccount(ctx context.Context, resourceGroupName, accountName string) error {
	accountsClient, err := getAccountsClient()
	if err != nil {
		return err
	}

	future, err := accountsClient.BeginDelete(
		ctx,
		resourceGroupName,
		accountName,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cannot delete account: %v", err)
	}

	_, err = future.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot get the account delete future response: %v", err)
	}

	return nil
}

// WaitForNoANFResource waits for a specified resource to don't exist anymore following a deletion.
// This is due to a known issue related to ARM Cache where the state of the resource is still cached within ARM infrastructure
// reporting that it still exists so looping into a get process will return 404 as soon as the cached state expires
func WaitForNoANFResource(ctx context.Context, resourceID string, intervalInSec int, retries int, checkForReplication bool) error {
	var err error

	for i := 0; i < retries; i++ {
		time.Sleep(time.Duration(intervalInSec) * time.Second)
		if uri.IsANFSnapshot(resourceID) {
			client, _ := getSnapshotsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				uri.GetANFVolume(resourceID),
				uri.GetANFSnapshot(resourceID),
				nil,
			)
		} else if uri.IsANFVolume(resourceID) {
			client, _ := getVolumesClient()
			if !checkForReplication {
				_, err = client.Get(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
					nil,
				)
			} else {
				_, err = client.ReplicationStatus(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
					nil,
				)
			}
		} else if uri.IsANFCapacityPool(resourceID) {
			client, _ := getPoolsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				nil,
			)
		} else if uri.IsANFSnapshotPolicy(resourceID) {
			client, _ := getSnapshotPoliciesClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFSnapshotPolicy(resourceID),
				nil,
			)
		} else if uri.IsANFAccount(resourceID) {
			client, _ := getAccountsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				nil,
			)
		}

		// In this case error is expected
		if err != nil {
			return nil
		}
	}

	return fmt.Errorf("exceeded number of retries: %v", retries)
}

// WaitForANFResource waits for a specified resource to be fully ready following a creation operation.
func WaitForANFResource(ctx context.Context, resourceID string, intervalInSec int, retries int, checkForReplication bool) error {
	var err error

	for i := 0; i < retries; i++ {
		time.Sleep(time.Duration(intervalInSec) * time.Second)
		if uri.IsANFSnapshot(resourceID) {
			client, _ := getSnapshotsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				uri.GetANFVolume(resourceID),
				uri.GetANFSnapshot(resourceID),
				nil,
			)
		} else if uri.IsANFVolume(resourceID) {
			client, _ := getVolumesClient()
			if !checkForReplication {
				_, err = client.Get(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
					nil,
				)
			} else {
				_, err = client.ReplicationStatus(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
					nil,
				)
			}
		} else if uri.IsANFCapacityPool(resourceID) {
			client, _ := getPoolsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				nil,
			)
		} else if uri.IsANFSnapshotPolicy(resourceID) {
			client, _ := getSnapshotPoliciesClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFSnapshotPolicy(resourceID),
				nil,
			)
		} else if uri.IsANFAccount(resourceID) {
			client, _ := getAccountsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				nil,
			)
		}

		// In this case, we exit when there is no error
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("resource still not found after number of retries: %v, error: %v", retries, err)
}
