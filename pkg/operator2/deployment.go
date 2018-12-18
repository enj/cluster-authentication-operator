package operator2

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *osinOperator) getGeneration() int64 {
	deployment, err := c.deployments.Deployments(targetName).Get(targetName, metav1.GetOptions{})
	if err != nil {
		return -1
	}
	return deployment.Generation
}

func defaultDeployment(resourceVersions ...string) *appsv1.Deployment {
	labels := defaultLabels()
	replicas := int32(3) // TODO configurable?
	gracePeriod := int64(30)

	deployment := &appsv1.Deployment{
		ObjectMeta: defaultMeta(),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   targetName,
					Labels: labels,
					Annotations: map[string]string{
						"osin.openshift.io/rv": strings.Join(resourceVersions, ","),
					},
				},
				Spec: corev1.PodSpec{
					// we want to deploy on master nodes
					NodeSelector: map[string]string{
						// empty string is correct
						"node-role.kubernetes.io/master": "",
					},
					Affinity: &corev1.Affinity{
						// spread out across master nodes rather than congregate on one
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{{
								Weight: 100,
								PodAffinityTerm: corev1.PodAffinityTerm{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: defaultLabels(),
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							}},
						},
					},
					// toleration is a taint override. we can and should be scheduled on a master node.
					Tolerations: []corev1.Toleration{
						{
							Key:      "node-role.kubernetes.io/master",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
					},
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 corev1.DefaultSchedulerName,
					TerminationGracePeriodSeconds: &gracePeriod,
					SecurityContext:               &corev1.PodSecurityContext{},
					Containers: []corev1.Container{
						consoleContainer(cr),
					},
					Volumes: consoleVolumes(volumeConfigList),
				},
			},
		},
	}
	return deployment
}
