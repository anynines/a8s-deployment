# API Reference

## Packages
- [servicebindings.anynines.com/v1alpha1](#servicebindingsanyninescomv1alpha1)

## servicebindings.anynines.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the servicebindings v1alpha1 API group

### Resource Types
- [ServiceBinding](#servicebinding)
- [ServiceBindingList](#servicebindinglist)

#### InstanceRef

InstanceRef is a reference to a Data Service Instance.

_Appears in:_
- [ServiceBindingSpec](#servicebindingspec)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | APIVersion is the <api_group>/<version> of the referenced Data Service Instance, e.g. "postgresql.anynines.com/v1alpha1" or "redis.anynines.com/v1alpha1". |
| `kind` _string_ | Kind is the Kubernetes API Kind of the referenced Data Service Instance. |
| `NamespacedName` _[NamespacedName](#namespacedname)_ | NamespacedName is the Kubernetes API Kind of the referenced Data Service Instance. |

#### NamespacedName

NamespacedName represents a Kubernetes API namespace and name. It's factored out to its own type for reusability.

_Appears in:_
- [InstanceRef](#instanceref)
- [ServiceBindingStatus](#servicebindingstatus)

| Field | Description |
| --- | --- |
| `namespace` _string_ |  |
| `name` _string_ |  |

#### ServiceBinding

ServiceBinding is the Schema for the servicebindings API

_Appears in:_
- [ServiceBindingList](#servicebindinglist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1alpha1`
| `kind` _string_ | `ServiceBinding`
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[ServiceBindingSpec](#servicebindingspec)_ |  |
| `status` _[ServiceBindingStatus](#servicebindingstatus)_ |  |

#### ServiceBindingList

ServiceBindingList contains a list of ServiceBinding

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `servicebindings.anynines.com/v1alpha1`
| `kind` _string_ | `ServiceBindingList`
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[ServiceBinding](#servicebinding) array_ |  |

#### ServiceBindingSpec

ServiceBindingSpec defines the desired state of the ServiceBinding

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `instance` _[InstanceRef](#instanceref)_ | Instance identifies the Data Service Instance that the ServiceBinding binds to. |

#### ServiceBindingStatus

ServiceBindingStatus defines the observed state of the ServiceBinding

_Appears in:_
- [ServiceBinding](#servicebinding)

| Field | Description |
| --- | --- |
| `secret` _[NamespacedName](#namespacedname)_ | Secret contains the namespace and name of the Kubernetes API secret that stores the credentials and information (e.g. URL) associated to the service binding to access the bound Data Service Instance. |
| `implemented` _boolean_ | Implemented is `true` if and only if the service binding has been implemented by creating a user with the appropriate permissions in the bound Data Service Instance. Users can safely consume the service binding secret identified by `Secret` IF AND ONLY IF `Implemented` is true. In other words, even if the secret identified by `Secret` gets created before `Implemented` becomes true, users MUST NOT consume that secret before `Implemented` has become true. |
| `error` _string_ | Error is a message explaining why the service binding could not be implemented if that's the case. |

