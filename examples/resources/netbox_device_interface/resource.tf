// Assumes a device with ID 123 exists
resource "netbox_device_interface" "test" {
  name      = "testinterface"
  device_id = 123
  type      = "1000base-t"
}

// Custom fields must be defined for dcim.interface before assigning values.
resource "netbox_custom_field" "dhcp_helper" {
  name          = "dhcp_helper"
  type          = "text"
  content_types = ["dcim.interface"]
}

resource "netbox_device_interface" "svi" {
  name      = "Vlan100"
  device_id = 123
  type      = "virtual"

  custom_fields = {
    (netbox_custom_field.dhcp_helper.name) = "192.0.2.10"
  }
}
