# from geopy.geocoders import Nominatim
from OSMPythonTools.nominatim import Nominatim
from geopy.extra.rate_limiter import RateLimiter
import json
import ast
import time


def get_reverse_geocode(latitude, longitude):
    # Initialize the API
    print('latitude and longitude are: ' + latitude + ', ' + longitude)

    # Uses local nominatim server
    nominatim = Nominatim(endpoint='http://vibecheck.tech:8080/')
    location = nominatim.query(latitude, longitude, reverse=True, zoom=10).address()

    # geolocator = Nominatim(user_agent='myGeocoder')
    # location = geolocator.reverse(latitude+","+longitude, timeout=10)
    # time.sleep(1)
    # reverse = RateLimiter(geolocator.reverse, min_delay_seconds=1)
    # location = reverse((latitude, longitude), language='en', exactly_one=True)
    # print('location.raw is:')
    # print(location.raw)
    locationEntry = {'city': '', 'state': ''}
    jsonReturn = {'reverseGeocodeResult': ''}
    # Access the values in the dictionary
    town = ''
    city = ''
    state = ''
    try:
        # print('location town is: ' + str(location.raw['address']['town']))
        town = str(location['town'])
    except:
        print('vibecheck-locations-api: handling exception, OSM tag `town` not in entry')

    try:
        # print('location city is: ' + str(location.raw['address']['city']))
        # if str(location['city'] == 'Philadelphia'):
        city = str(location['city'])
    except:
        print('vibecheck-locations-api: handling exception, OSM tag `city` not in entry')

    try:
        # print('location city is: ' + str(location.raw['address']['city']))
        state = str(location['state'])
    except:
        print('vibecheck-locations-api: handling exception, OSM tag `state` not in entry')

    if location is None:
        jsonReturn['reverseGeocodeResult'] = 'Service only in PA'
        return jsonReturn

    if city == '' and town == '' and state == '':
        jsonReturn['reverseGeocodeResult'] = 'Error'
        return jsonReturn

    if town != '' and city != '':
        locationEntry['city'] = city
    elif town == '' and city != '':
        locationEntry['city'] = city
    else:
        locationEntry['city'] = town
    locationEntry['state'] = state
    if locationEntry['city'] == '' and locationEntry['state'] == '':
        jsonReturn['reverseGeocodeResult'] = latitude + \
            ', ' + longitude  # should be okay
    elif locationEntry['city'] == '' and locationEntry['state'] != '':
        jsonReturn['reverseGeocodeResult'] = locationEntry['state']
    elif locationEntry['city'] != '' and locationEntry['state'] == '':
        jsonReturn['reverseGeocodeResult'] = locationEntry['city']
    # city tag actually has state included in it
    elif ',' in locationEntry['city']:
        jsonReturn['reverseGeocodeResult'] = locationEntry['city']
    # state tag actually has city included in it
    elif ',' in locationEntry['state']:
        jsonReturn['reverseGeocodeResult'] = locationEntry['state']
    else:
        jsonReturn['reverseGeocodeResult'] = locationEntry['city'] + \
            ', ' + locationEntry['state']
    return jsonReturn
