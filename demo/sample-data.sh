#!/usr/bin/env bash

set -e

NETBOX_API=http://localhost:8000
NETBOX_TOKEN=example-token
BULK_LOAGEN_API=http://localhost:8080/api/v1

netbox() {
  res=$(curl -S -s -H "Content-Type: application/json" -H "Accept: application/json" -H "Authorization: Token $NETBOX_TOKEN" "$@")
  code=$?

  if [[ "$code" -ne 0 ]] && [[ "$res" != *"already exists"* ]] && [[ "$res" != *"Device name must be unique per site."* ]]; then
    echo "$res" | jq -r '.'
    exit 1
  fi

  if [[ "$res" == *"already exists"* ]] || [[ "$res" == *"Device name must be unique per site."* ]]; then
    return 0
  fi

  echo -n "$res"
}

echo "Creating site"
netbox -X POST -d '{"name":"DC01","slug":"dc01","facility":"DC01:23"}' "$NETBOX_API/api/dcim/sites/"

echo "Creating rack if not exists"
netbox -X GET "$NETBOX_API/api/dcim/racks/?name=Rack01" -s | jq -e '.count == 0' >/dev/null && netbox -X POST -d '{"name":"Rack01","site":{"name":"DC01"},"u_height":48,"facility_id":"R1234"}' "$NETBOX_API/api/dcim/racks/"

echo "Creating manufacturer"
netbox -X POST -d '{"name":"Dummy","slug":"dummy"}' "$NETBOX_API/api/dcim/manufacturers/"

echo "Creating device role"
netbox -X POST -d '{"name":"Patchpanel","slug":"patchpanel"}' "$NETBOX_API/api/dcim/device-roles/"

echo "Creating device type"
netbox -X POST -d '{"model":"Patchpanel","slug":"patchpanel","manufacturer":{"name":"Dummy"}}' "$NETBOX_API/api/dcim/device-types/"

echo "Getting device type id"
device_type_id=$(netbox -X GET "$NETBOX_API/api/dcim/device-types/?model=Patchpanel" | jq -r '.results[0].id')
echo -e "Device type id: $device_type_id"

echo "Creating rear port templates"
for i in $(seq 1 2 23); do
  i2=$(printf "%02d" $(("$i" + 1)))
  i=$(printf "%02d" "$i")
  netbox -X POST -d '{"device_type":'"$device_type_id"',"type":"lc-pc","name":"Fiber '"$i"'/'"$i2"'"}' "$NETBOX_API/api/dcim/rear-port-templates/"
done

echo -e "Creating front port templates"
for i in $(seq 1 2 23); do
  i2=$(printf "%02d" $(("$i" + 1)))
  i=$(printf "%02d" "$i")
  netbox -X POST -d '{"device_type":'"$device_type_id"',"type":"lc-pc","name":"Fiber '"$i"'/'"$i2"'","rear_port":{"name":"Fiber '"$i"'/'"$i2"'"}}' "$NETBOX_API/api/dcim/front-port-templates/"
done

echo "Creating device"
netbox -X POST -d '{"name":"Demarc Panel","device_type":'"$device_type_id"',"role":{"name":"Patchpanel"},"site":{"name":"DC01"},"rack":{"name":"Rack01"},"position":48,"face":"front"}' "$NETBOX_API/api/dcim/devices/"

echo "Getting device id"
device_id=$(netbox -X GET "$NETBOX_API/api/dcim/devices/?site=dc01&role=patchpanel" -s | jq -r '.results[0].id')
echo "Device id: $device_id"

echo "Creating custom link on rear ports"
netbox -X POST -d '{"name":"LOA","content_types":["dcim.rearport"],"link_text":"LOA","link_url":"'"$BULK_LOAGEN_API"'/devices/{{object.device.id}}?rear_port={{object.id}}","new_window":true}' "$NETBOX_API/api/extras/custom-links/"

echo "Showing custom link in rear port table"
netbox -o /dev/null -X PATCH -d '{"tables":{"DeviceRearPortTable":{"columns":["name","label","type","positions","description","cable","link_peer","cl_LOA"],"ordering":["name"]}}}' "$NETBOX_API/api/users/config/"

echo "Creating custom link on devices"
netbox -X POST -d '{"name":"LOA from Patchpanel","content_types":["dcim.device"],"link_text":"LOA","link_url":"'"$BULK_LOAGEN_API"'/devices/{{object.id}}","new_window":true}' "$NETBOX_API/api/extras/custom-links/"

echo "Showing custom link in device table"
netbox -o /dev/null -X PATCH -d '{"tables":{"DeviceTable":{"columns":["name","status","tenant","site","location","rack","role","manufacturer","device_type","primary_ip","cl_LOA from Patchpanel"],"ordering":["name"]}}}' "$NETBOX_API/api/users/config/"

echo ""
echo -e "You can visit $NETBOX_API/dcim/devices/$device_id/ and click on the LOA link\nor $NETBOX_API/dcim/devices/$device_id/rear-ports/ and click on the LOA link at the correct rear port."
