package mapper

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	infrav1alpha1 "infra-operator/api/v1alpha1"
	"infra-operator/internal/domain/apigateway"
)

func CRToDomainAPIGateway(cr *infrav1alpha1.APIGateway) *apigateway.API {
	api := &apigateway.API{
		Name:                      cr.Spec.Name,
		Description:               cr.Spec.Description,
		ProtocolType:              cr.Spec.ProtocolType,
		EndpointType:              cr.Spec.EndpointType,
		DisableExecuteApiEndpoint: cr.Spec.DisableExecuteApiEndpoint,
		Tags:                      cr.Spec.Tags,
		DeletionPolicy:            cr.Spec.DeletionPolicy,
	}

	if cr.Spec.CorsConfiguration != nil {
		api.CorsConfiguration = &apigateway.CorsConfiguration{
			AllowOrigins:     cr.Spec.CorsConfiguration.AllowOrigins,
			AllowMethods:     cr.Spec.CorsConfiguration.AllowMethods,
			AllowHeaders:     cr.Spec.CorsConfiguration.AllowHeaders,
			ExposeHeaders:    cr.Spec.CorsConfiguration.ExposeHeaders,
			MaxAge:           cr.Spec.CorsConfiguration.MaxAge,
			AllowCredentials: cr.Spec.CorsConfiguration.AllowCredentials,
		}
	}

	if cr.Status.APIID != "" {
		api.APIID = cr.Status.APIID
		api.APIEndpoint = cr.Status.APIEndpoint
	}

	return api
}

func DomainToStatusAPIGateway(api *apigateway.API, cr *infrav1alpha1.APIGateway) {
	cr.Status.Ready = api.IsReady()
	cr.Status.APIID = api.APIID
	cr.Status.APIEndpoint = api.APIEndpoint
	cr.Status.ProtocolType = api.ProtocolType

	now := metav1.NewTime(time.Now())
	cr.Status.LastSyncTime = &now
}
