/*
Copyright © 2020 The k3d Author(s)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package v1alpha2

// JSONSchema describes the schema used to validate config files
/* TODO: JSONSchema should be an embedded file. We're moving to the //go:embed tag as of Go 1.16
 * ... and only hardcode it here to avoid using 3rd party tools like go-bindata or packr right now for the time being
 */
var JSONSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "SimpleConfig",
  "type": "object",
  "required": [
    "apiVersion",
    "kind"
  ],
  "properties": {
    "apiVersion": {
      "type": "string",
      "enum": [
        "k3d.io/v1alpha2"
      ]
    },
    "kind": {
      "type": "string",
      "enum": [
        "Simple"
      ]
    },
    "name": {
      "description": "Name of the cluster (must be a valid hostname and will be prefixed with 'k3d-'). Example: 'mycluster'.",
			"type": "string",
			"format": "hostname"
    },
    "servers": {
      "type": "number",
      "minimum": 1
    },
    "agents": {
      "type": "number",
      "minimum": 0
    },
    "kubeAPI": {
      "type": "object",
      "properties": {
        "host": {
          "type": "string",
          "format": "hostname"
        },
        "hostIP": {
          "type": "string",
          "format": "ipv4"
        },
        "hostPort": {
          "type":"string"
        }
      },
      "additionalProperties": false
    },
    "image": {
      "type": "string"
    },
    "network": {
      "type": "string"
    },
    "token": {
      "type": "string"
    },
    "volumes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "volume": {
            "type": "string"
          },
          "nodeFilters": {  
            "$ref": "#/definitions/nodeFilters"
          }
        },
        "additionalProperties": false
      }
    },
    "ports": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "port": {
            "type": "string"
          },
          "nodeFilters": {  
            "$ref": "#/definitions/nodeFilters"
          }
        },
        "additionalProperties": false
      }
    },
    "labels": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "label": {
            "type": "string"
          },
          "nodeFilters": {  
            "$ref": "#/definitions/nodeFilters"
          }
        },
        "additionalProperties": false
      }
    },
    "options": {
      "type": "object",
      "properties": {
        "k3d": {
          "type": "object",
          "properties": {
            "wait": {
              "type": "boolean",
              "default": true
            },
            "timeout": {
              "type": "string"
            },
            "disableLoadbalancer": {
              "type": "boolean",
              "default": false
            },
            "disableImageVolume": {
              "type": "boolean",
              "default": false
            },
            "disableRollback": {
              "type": "boolean",
              "default": false
            },
            "disableHostIPInjection": {
              "type": "boolean",
              "default": false
            }
          },
          "additionalProperties": false
        },
        "k3s": {
          "type": "object",
          "properties": {
            "extraServerArgs": {
              "type": "array",
              "items": {
                "type": "string"
              }
            },
            "extraAgentArgs": {
              "type": "array",
              "items": {
                "type": "string"
              }
            }
          },
          "additionalProperties": false
        },
        "kubeconfig": {
          "type": "object",
          "properties": {
            "updateDefaultKubeconfig": {
              "type": "boolean",
              "default": true
            },
            "switchCurrentContext": {
              "type": "boolean",
              "default": true
            }
          },
          "additionalProperties": false
        },
        "runtime": {
          "type": "object",
          "properties": {
            "gpuRequest": {
              "type": "string"
            }
          }
        }
      },
      "additionalProperties": false
    },
    "env": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "envVar": {
            "type": "string"
          },
          "nodeFilters": {  
            "$ref": "#/definitions/nodeFilters"
          }
        },
        "additionalProperties": false
      }
    },
    "registries": {
      "type": "object"
    }
  },
  "additionalProperties": false,
  "definitions": {
    "nodeFilters": {  
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  }
}`
