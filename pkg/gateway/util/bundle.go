package util

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
)

type Bundle struct {
	XMLName    xml.Name   `xml:"l7:Bundle"`
	XMLNS      string     `xml:"xmlns:l7,attr"`
	References References `xml:"l7:References"`
	Mappings   Mappings   `xml:"l7:Mappings"`
}

type References struct {
	Item []Item `xml:"l7:Item"`
}

type Item struct {
	Name     string   `xml:"l7:Name"`
	ID       string   `xml:"l7:Id"`
	Type     string   `xml:"l7:Type"`
	Resource Resource `xml:"l7:Resource"`
}

type Resource struct {
	ClusterProperty *ClusterProperty `xml:"l7:ClusterProperty,omitempty"`
	ListenPort      *ListenPort      `xml:"l7:ListenPort,omitempty"`
}

type ClusterProperty struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"l7:Name"`
	Value string `xml:"l7:Value"`
}

type Mappings struct {
	Mapping []Mapping `xml:"l7:Mapping"`
}
type Mapping struct {
	Action     string     `xml:"action,attr"`
	SrcId      string     `xml:"srcId,attr"`
	Type       string     `xml:"type,attr"`
	Properties Properties `xml:"l7:Properties"`
}

type Properties struct {
	Property []Property `xml:"l7:Property"`
}

type Property struct {
	Key          string `xml:"key,attr"`
	StringValue  string `xml:"l7:StringValue,omitempty"`
	BooleanValue bool   `xml:"l7:BooleanValue,omitempty"`
}

func randToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

type ListenPort struct {
	ID              string          `xml:"id,attr"`
	Name            string          `xml:"l7:Name"`
	Enabled         string          `xml:"l7:Enabled"`
	Protocol        string          `xml:"l7:Protocol"`
	Port            string          `xml:"l7:Port"`
	EnabledFeatures EnabledFeatures `xml:"l7:EnabledFeatures"`
	TlsSettings     *TlsSettings    `xml:"l7:TlsSettings"`
}

type TlsSettings struct {
	ClientAuthentication string              `xml:"l7:ClientAuthentication"`
	EnabledVersions      EnabledVersions     `xml:"l7:EnabledVersions"`
	EnabledCipherSuites  EnabledCipherSuites `xml:"l7:EnabledCipherSuites"`
	UseCipherSuitesOrder bool                `xml:"l7:UseCipherSuitesOrder"`
	Properties           Properties          `xml:"l7:Properties"`
}

type EnabledVersions struct {
	StringValue []string `xml:"l7:StringValue"`
}

type EnabledCipherSuites struct {
	StringValue []string `xml:"l7:StringValue"`
}

type EnabledFeatures struct {
	StringValue []string `xml:"l7:StringValue"`
}

func BuildCWPBundle(cwps map[string]string) ([]byte, error) {
	refs := References{}
	items := []Item{}
	mapping := []Mapping{}

	for cwp, val := range cwps {
		randomId, err := randToken(16)

		if err != nil {
			return nil, err
		}

		resource := Resource{ClusterProperty: &ClusterProperty{
			ID:    randomId,
			Name:  cwp,
			Value: val,
		}}
		items = append(items, Item{Name: cwp,
			ID:       randomId,
			Type:     "CLUSTER_PROPERTY",
			Resource: resource,
		})

		properties := []Property{{
			Key:         "MapBy",
			StringValue: "name",
		}, {
			Key:         "MapTo",
			StringValue: cwp,
		},
		}

		mapping = append(mapping, Mapping{
			Action:     "NewOrUpdate",
			SrcId:      randomId,
			Type:       "CLUSTER_PROPERTY",
			Properties: Properties{Property: properties},
		})

		refs.Item = items
	}

	mappings := Mappings{Mapping: mapping}

	bundle := Bundle{
		XMLNS:      "http://ns.l7tech.com/2010/04/gateway-management",
		References: refs,
		Mappings:   mappings,
	}

	bundleBytes, err := xml.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	return bundleBytes, nil

}

func BuildListenPortBundle(cipherSuites []string, tlsVersions []string) ([]byte, error) {
	trafficId, _ := randToken(16)
	managementId, _ := randToken(16)
	plaintextId, _ := randToken(16)
	internodeCommunicationId, _ := randToken(16)

	refs := References{}
	items := []Item{}
	mapping := []Mapping{}

	plantextPort := Item{
		Name: "Default HTTP (8080)",
		ID:   plaintextId,
		Type: "SSG_CONNECTOR",
		Resource: Resource{
			ListenPort: &ListenPort{
				ID:       plaintextId,
				Name:     "Default HTTP (8080)",
				Enabled:  "false",
				Protocol: "HTTP",
				Port:     "8080",
				EnabledFeatures: EnabledFeatures{
					StringValue: []string{
						"Published service message input",
					}},
			},
		},
	}

	items = append(items, plantextPort)

	managementPort := Item{
		Name: "Default HTTPS (9443)",
		ID:   trafficId,
		Type: "SSG_CONNECTOR",
		Resource: Resource{
			ListenPort: &ListenPort{
				ID:       managementId,
				Name:     "Default HTTPS (9443)",
				Enabled:  "true",
				Protocol: "HTTPS",
				Port:     "9443",
				EnabledFeatures: EnabledFeatures{
					StringValue: []string{
						"Published service message input",
						"Administrative access",
						"Browser-based administration",
						"Built-in services",
					}},
				TlsSettings: &TlsSettings{
					ClientAuthentication: "Optional",
					EnabledVersions: EnabledVersions{
						StringValue: tlsVersions,
					},
					EnabledCipherSuites: EnabledCipherSuites{
						StringValue: cipherSuites,
					},
					UseCipherSuitesOrder: true,
					Properties: Properties{
						Property: []Property{
							{
								Key:          "usesTLS",
								BooleanValue: true,
							},
						},
					},
				},
			},
		},
	}

	items = append(items, managementPort)

	trafficPort := Item{
		Name: "Default HTTPS (8443)",
		ID:   trafficId,
		Type: "SSG_CONNECTOR",
		Resource: Resource{
			ListenPort: &ListenPort{
				ID:       trafficId,
				Name:     "Default HTTPS (8443)",
				Enabled:  "true",
				Protocol: "HTTPS",
				Port:     "8443",
				EnabledFeatures: EnabledFeatures{
					StringValue: []string{
						"Published service message input",
					}},
				TlsSettings: &TlsSettings{
					ClientAuthentication: "Optional",
					EnabledVersions: EnabledVersions{
						StringValue: tlsVersions,
					},
					EnabledCipherSuites: EnabledCipherSuites{
						StringValue: cipherSuites,
					},
					UseCipherSuitesOrder: true,
					Properties: Properties{
						Property: []Property{
							{
								Key:          "usesTLS",
								BooleanValue: true,
							},
						},
					},
				},
			},
		},
	}

	items = append(items, trafficPort)

	internodeCommunicationPort := Item{
		Name: "Node HTTPS (2124)",
		ID:   internodeCommunicationId,
		Type: "SSG_CONNECTOR",
		Resource: Resource{
			ListenPort: &ListenPort{
				ID:       internodeCommunicationId,
				Name:     "Node HTTPS (2124)",
				Enabled:  "false",
				Protocol: "HTTPS",
				Port:     "2124",
				EnabledFeatures: EnabledFeatures{
					StringValue: []string{
						"Inter-Node Communication",
					}},
				TlsSettings: &TlsSettings{
					ClientAuthentication: "Optional",
					EnabledVersions: EnabledVersions{
						StringValue: []string{
							"TLSv1.2",
							"TLSv1.3",
						},
					},
					EnabledCipherSuites: EnabledCipherSuites{
						StringValue: []string{
							"TLS_DHE_RSA_WITH_AES_256_CBC_SHA",
							"TLS_RSA_WITH_AES_256_CBC_SHA",
							"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
							"TLS_DHE_RSA_WITH_AES_128_CBC_SHA",
							"TLS_RSA_WITH_AES_128_CBC_SHA",
							"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
						},
					},
					UseCipherSuitesOrder: true,
					Properties: Properties{
						Property: []Property{
							{
								Key:          "usesTLS",
								BooleanValue: true,
							},
						},
					},
				},
			},
		},
	}

	items = append(items, internodeCommunicationPort)

	refs.Item = items

	mapping = append(mapping, Mapping{
		Action: "NewOrUpdate",
		SrcId:  plaintextId,
		Type:   "SSG_CONNECTOR",
		Properties: Properties{Property: []Property{{
			Key:         "MapBy",
			StringValue: "name",
		}, {
			Key:         "MapTo",
			StringValue: "Default HTTP (8080)",
		},
		}},
	})

	mapping = append(mapping, Mapping{
		Action: "NewOrUpdate",
		SrcId:  managementId,
		Type:   "SSG_CONNECTOR",
		Properties: Properties{Property: []Property{{
			Key:         "MapBy",
			StringValue: "name",
		}, {
			Key:         "MapTo",
			StringValue: "Default HTTPS (9443)",
		},
		}},
	})

	mapping = append(mapping, Mapping{
		Action: "NewOrUpdate",
		SrcId:  trafficId,
		Type:   "SSG_CONNECTOR",
		Properties: Properties{Property: []Property{{
			Key:         "MapBy",
			StringValue: "name",
		}, {
			Key:         "MapTo",
			StringValue: "Default HTTPS (8443)",
		},
		}},
	})

	mapping = append(mapping, Mapping{
		Action: "NewOrUpdate",
		SrcId:  internodeCommunicationId,
		Type:   "SSG_CONNECTOR",
		Properties: Properties{Property: []Property{{
			Key:         "MapBy",
			StringValue: "name",
		}, {
			Key:         "MapTo",
			StringValue: "Node HTTPS (2124)",
		},
		}},
	})

	mappings := Mappings{Mapping: mapping}

	bundle := Bundle{
		XMLNS:      "http://ns.l7tech.com/2010/04/gateway-management",
		References: refs,
		Mappings:   mappings,
	}

	bundleBytes, err := xml.Marshal(bundle)
	if err != nil {
		return nil, err
	}

	return bundleBytes, nil
}
