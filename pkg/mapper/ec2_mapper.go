package mapper

import (
	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/ec2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CRToDomainEC2Instance(cr *infrav1alpha1.EC2Instance) *ec2.Instance {
	instance := &ec2.Instance{
		InstanceName:          cr.Spec.InstanceName,
		InstanceType:          cr.Spec.InstanceType,
		ImageID:               cr.Spec.ImageID,
		KeyName:               cr.Spec.KeyName,
		SubnetID:              cr.Spec.SubnetID,
		SecurityGroupIDs:      cr.Spec.SecurityGroupIDs,
		IAMInstanceProfile:    cr.Spec.IAMInstanceProfile,
		UserData:              cr.Spec.UserData,
		Monitoring:            cr.Spec.Monitoring,
		DisableAPITermination: cr.Spec.DisableAPITermination,
		EBSOptimized:          cr.Spec.EBSOptimized,
		Tags:                  cr.Spec.Tags,
		DeletionPolicy:        cr.Spec.DeletionPolicy,
	}

	// Convert block device mappings
	if len(cr.Spec.BlockDeviceMappings) > 0 {
		instance.BlockDeviceMappings = make([]ec2.BlockDeviceMapping, 0, len(cr.Spec.BlockDeviceMappings))
		for _, bdm := range cr.Spec.BlockDeviceMappings {
			mapping := ec2.BlockDeviceMapping{
				DeviceName: bdm.DeviceName,
			}
			if bdm.EBS != nil {
				mapping.EBS = &ec2.EBSBlockDevice{
					VolumeSize:          bdm.EBS.VolumeSize,
					VolumeType:          bdm.EBS.VolumeType,
					IOPS:                bdm.EBS.IOPS,
					DeleteOnTermination: bdm.EBS.DeleteOnTermination,
					Encrypted:           bdm.EBS.Encrypted,
					KMSKeyID:            bdm.EBS.KMSKeyID,
				}
			}
			instance.BlockDeviceMappings = append(instance.BlockDeviceMappings, mapping)
		}
	}

	// If status has InstanceID, use it
	if cr.Status.InstanceID != "" {
		instance.InstanceID = cr.Status.InstanceID
	}

	return instance
}

func DomainToStatusEC2Instance(instance *ec2.Instance, cr *infrav1alpha1.EC2Instance) {
	cr.Status.Ready = instance.IsRunning()
	cr.Status.InstanceID = instance.InstanceID
	cr.Status.InstanceState = instance.InstanceState
	cr.Status.PrivateIP = instance.PrivateIP
	cr.Status.PublicIP = instance.PublicIP
	cr.Status.PrivateDNS = instance.PrivateDNS
	cr.Status.PublicDNS = instance.PublicDNS
	cr.Status.AvailabilityZone = instance.AvailabilityZone

	if instance.LaunchTime != nil {
		cr.Status.LaunchTime = &metav1.Time{Time: *instance.LaunchTime}
	}
	if instance.LastSyncTime != nil {
		cr.Status.LastSyncTime = &metav1.Time{Time: *instance.LastSyncTime}
	}
}
