package auspost

var DemoDeliveryEvents = `
{
  "tracking_results": [
    {
      "tracking_id": "%s",
      "status": "Delivered",
      "trackable_items": [
        {
          "article_id": "%s",
          "product_type": "Parcel Post",
          "events": [
            {
              "description": "Delivered",
              "date": "2022-12-30T16:32:17+11:00"
            },
            {
              "location": "INALA QLD",
              "description": "Awaiting collection at INALA HEIGHTS LPO",
              "date": "2022-12-30T08:23:17+11:00"
            },
            {
              "location": "HEATHWOOD QLD",
              "description": "In transit",
              "date": "2022-12-29T14:46:51+11:00"
            },
            {
              "location": "RICHLANDS QLD",
              "description": "Attempted delivery - Unable to gain access",
              "date": "2022-12-29T13:44:30+11:00"
            },
            {
              "location": "HEATHWOOD QLD",
              "description": "Onboard for delivery",
              "date": "2022-12-29T08:25:37+11:00"
            },
            {
              "description": "In transit to next facility in DARRA QLD",
              "date": "2022-12-29T01:15:01+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Item processed at facility",
              "date": "2022-12-29T00:39:11+11:00"
            },
            {
              "description": "In transit to next facility in REDBANK QLD",
              "date": "2022-12-28T15:20:57+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Item processed at facility",
              "date": "2022-12-28T14:57:14+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Arrived at sorting facility",
              "date": "2022-12-28T11:05:53+11:00"
            },
            {
              "description": "In transit to next facility in REDBANK QLD",
              "date": "2022-12-26T14:04:56+11:00"
            },
            {
              "location": "GRANVILLE NSW",
              "description": "Item processed at facility",
              "date": "2022-12-26T13:56:02+11:00"
            }
          ],
          "status": "Delivered"
        }
      ]
    }
  ]
}
`

var DemoInTransitEvents = `
{
  "tracking_results": [
    {
      "tracking_id": "%s",
      "status": "In transit",
      "trackable_items": [
        {
          "article_id": "%s",
          "product_type": "Parcel Post",
          "events": [
            {
              "location": "HEATHWOOD QLD",
              "description": "In transit",
              "date": "2022-12-29T14:46:51+11:00"
            },
            {
              "location": "RICHLANDS QLD",
              "description": "Attempted delivery - Unable to gain access",
              "date": "2022-12-29T13:44:30+11:00"
            },
            {
              "location": "HEATHWOOD QLD",
              "description": "Onboard for delivery",
              "date": "2022-12-29T08:25:37+11:00"
            },
            {
              "description": "In transit to next facility in DARRA QLD",
              "date": "2022-12-29T01:15:01+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Item processed at facility",
              "date": "2022-12-29T00:39:11+11:00"
            },
            {
              "description": "In transit to next facility in REDBANK QLD",
              "date": "2022-12-28T15:20:57+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Item processed at facility",
              "date": "2022-12-28T14:57:14+11:00"
            },
            {
              "location": "REDBANK QLD",
              "description": "Arrived at sorting facility",
              "date": "2022-12-28T11:05:53+11:00"
            },
            {
              "description": "In transit to next facility in REDBANK QLD",
              "date": "2022-12-26T14:04:56+11:00"
            },
            {
              "location": "GRANVILLE NSW",
              "description": "Item processed at facility",
              "date": "2022-12-26T13:56:02+11:00"
            }
          ],
          "status": "In transit"
        }
      ]
    }
  ]
}
`
