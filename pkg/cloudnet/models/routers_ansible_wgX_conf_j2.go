// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

const wgX_conf_j2 = `
{% set interface = lookup('vars', 'wireguard_' + item + '_interface') -%}
{% set peers = lookup('vars', 'wireguard_' + item + '_peers') -%}

{% set interface_required_keys = { 'private_key': 'PrivateKey' } -%}
{% set interface_optional_keys = {
   'address': 'Address',
   'listen_port': 'ListenPort',
   'fw_mark': 'FwMark',
   'dns': 'DNS',
   'mtu': 'MTU',
   'table': 'Table',
   'pre_up': 'PreUp',
   'post_up': 'PostUp',
   'pre_down': 'PreDown',
   'post_down': 'PostDown',
   'save_config': 'SaveConfig'
} -%}
{% set peer_required_keys = {
   'public_key': 'PublicKey',
   'allowed_ips': 'AllowedIPs'
} -%}
{% set peer_optional_keys = {
   'endpoint': 'EndPoint',
   'preshared_key': 'PresharedKey',
   'persistent_keepalive': 'PersistentKeepalive'
} -%}
{{ ansible_managed | comment }}

[Interface]
{% for key, option in interface_required_keys.items() %}
{{ option }} = {{ interface[key] }}
{% endfor %}
{% for key, option in interface_optional_keys.items() %}
{% if interface[key] is defined %}
{{ option }} = {{ interface[key] }}
{% endif %}
{% endfor %}

{% for peer_name, peer in peers.items() %}
[Peer] # {{ peer_name }}
{% for key, option in peer_required_keys.items() %}
{{ option }} = {{ peer[key] }}
{% endfor %}
{% for key, option in peer_optional_keys.items() %}
{% if peer[key] is defined %}
{{ option }} = {{ peer[key] }}
{% endif %}
{% endfor %}

{% endfor %}
`
