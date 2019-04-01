// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * This file is part of the IoT Identity Service
 * Copyright 2019 Canonical Ltd.
 *
 * This program is free software: you can redistribute it and/or modify it
 * under the terms of the GNU Affero General Public License version 3, as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranties of MERCHANTABILITY,
 * SATISFACTORY QUALITY, or FITNESS FOR A PARTICULAR PURPOSE.
 * See the GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package memory

import (
	"fmt"
	"strconv"

	"github.com/CanonicalLtd/iot-identity/datastore"
	"github.com/CanonicalLtd/iot-identity/domain"
)

// Store implements an in-memory store for testing
type Store struct {
	Orgs []domain.Organization
	Roll []domain.Enrollment
}

// NewStore creates a new memory store
func NewStore() *Store {
	exOrg := domain.Organization{ID: "abc", Name: "Example Inc", RootKey: []byte(rootPEM), RootCert: []byte(certPEM)}
	dev1 := domain.Device{"example", "drone-1000", "DR1000A111", "", ""}
	dev2 := domain.Device{"example", "drone-1000", "DR1000B222", "example-store", "BBBBBBBBB"}

	return &Store{
		Orgs: []domain.Organization{exOrg},
		Roll: []domain.Enrollment{
			{
				ID:           "a111",
				Organization: exOrg,
				Device:       dev1,
				Status:       domain.StatusWaiting,
			},
			{
				ID:           "b222",
				Organization: exOrg,
				Device:       dev2,
				Status:       domain.StatusEnrolled,
			},
		},
	}
}

// OrganizationNew creates a new organization
func (mem *Store) OrganizationNew(organization datastore.OrganizationNewRequest) (string, error) {
	// Validate the organization
	if len(organization.Name) == 0 || len(organization.RootKey) == 0 || len(organization.RootCert) == 0 {
		return "", fmt.Errorf("the name and root CA details must be provided")
	}

	// Check we don't have it
	for _, org := range mem.Orgs {
		if org.Name == organization.Name {
			return "", fmt.Errorf("the organization `%s` already exists", organization.Name)
		}
	}

	// Store it
	id := strconv.Itoa(len(mem.Orgs) + 1)
	o := domain.Organization{
		ID:       id,
		Name:     organization.Name,
		RootKey:  organization.RootKey,
		RootCert: organization.RootCert,
	}
	mem.Orgs = append(mem.Orgs, o)
	return id, nil
}

// OrganizationGetByName fetches an organization by name
func (mem *Store) OrganizationGetByName(name string) (*domain.Organization, error) {
	for _, org := range mem.Orgs {
		if org.Name == name {
			return &org, nil
		}
	}
	return nil, fmt.Errorf("cannot find organization with name '%s'", name)
}

// OrganizationGet fetches an organization by ID
func (mem *Store) OrganizationGet(id string) (*domain.Organization, error) {
	for _, org := range mem.Orgs {
		if org.ID == id {
			return &org, nil
		}
	}
	return nil, fmt.Errorf("cannot find organization with ID '%s'", id)
}

// DeviceNew creates a new device registration
func (mem *Store) DeviceNew(device datastore.DeviceNewRequest) (string, error) {
	// Validate
	if len(device.Brand) == 0 || len(device.Model) == 0 || len(device.SerialNumber) == 0 || len(device.OrganizationID) == 0 {
		return "", fmt.Errorf("the provided device details are incomplete")
	}

	// Get the organization
	o, err := mem.OrganizationGet(device.OrganizationID)
	if err != nil {
		return "", err
	}

	// Check for duplicate
	for _, en := range mem.Roll {
		if en.Organization.ID == device.OrganizationID && en.Device.Brand == device.Brand && en.Device.Model == device.Model && en.Device.SerialNumber == device.SerialNumber {
			return "", fmt.Errorf("the device `%s/%s/%s` is already registered", device.Brand, device.Model, device.SerialNumber)
		}
	}

	// Store it
	id := strconv.Itoa(len(mem.Roll) + 1)
	d := domain.Device{
		Brand:        device.Brand,
		Model:        device.Model,
		SerialNumber: device.SerialNumber,
	}
	e := domain.Enrollment{
		ID:           id,
		Organization: *o,
		Device:       d,
		Credentials:  device.Credentials,
		Status:       domain.StatusWaiting,
	}
	mem.Roll = append(mem.Roll, e)
	return id, nil
}

// DeviceGet fetches a device registration
func (mem *Store) DeviceGet(brand, model, serial string) (*domain.Enrollment, error) { // Check for duplicate
	for _, en := range mem.Roll {
		if en.Device.Brand == brand && en.Device.Model == model && en.Device.SerialNumber == serial {
			return &en, nil
		}
	}
	return nil, fmt.Errorf("the device `%s/%s/%s` is not registered", brand, model, serial)
}

// DeviceEnroll enrols a device with the IoT service
func (mem *Store) DeviceEnroll(device datastore.DeviceEnrollRequest) (*domain.Enrollment, error) {
	// Get the registered device
	reg, err := mem.DeviceGet(device.Brand, device.Model, device.SerialNumber)
	if err != nil {
		return nil, err
	}

	// Update the registration to enroll the device
	reg.Device.DeviceKey = device.DeviceKey
	reg.Device.StoreID = device.StoreID
	reg.Status = domain.StatusEnrolled
	return reg, nil
}