package main

import (
	"sort"
)

//service areas
const (
	ServiceAreaEducationAndTraining = "Education and Training"
)

//association of service areas and their string representation
var serviceAreaStr = []string{
	"Miscellaneous", //default
	"Accounting, Auditing and Tax",
	"Advertising",
	"Alterations & Repairs",
	"Animal Care",
	"Architecture and Interior Design",
	"Banking and Finance",
	"Body Care and Massage",
	"Building and Construction",
	"Business Administration",
	"Car Cleaning and Repair",
	"Catering",
	"Child Care",
	"Computer and Electronics",
	"Customer Support",
	ServiceAreaEducationAndTraining,
	"Entertainment",
	"Event Planning and Management",
	"Fitness and Sports",
	"Furniture",
	"Gardening and Landscaping",
	"Graphic Design",
	"Hair and Skin Care",
	"Health and Wellness",
	"Home Appliances",
	"Home Cleaning",
	"Home Repair",
	"Insurance",
	"Legal",
	"Marketing",
	"Music",
	"Packaging and Delivery",
	"Photography",
	"Printing and Publishing",
	"Real Estate",
	"Senior Care",
	"Translation and Interpretation",
	"Transportation",
	"Travel",
	"Virtual Assistant",
	"Writing",
}

//ListServiceAreaStrs : list service area strings
func ListServiceAreaStrs() []string {
	l := make([]string, len(serviceAreaStr))
	i := 0
	for _, v := range serviceAreaStr {
		l[i] = v
		i++
	}
	sort.Strings(l)
	return l
}
