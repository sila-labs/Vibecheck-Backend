#######################################
# VIBECHECK LOCATION API
#######################################
from __future__ import print_function

from location_grab import get_locations
from reverse_geocode import get_reverse_geocode
from pydantic import BaseModel
from fastapi import FastAPI
from typing import Optional
from fastapi.encoders import jsonable_encoder
from fastapi.responses import JSONResponse
from pprint import pprint
from datetime import date, datetime, timedelta
# from fastapi_utils.session import FastAPISessionMaker
# from fastapi_utils.tasks import repeat_every
import urllib.parse
import urllib.request
import redis
import requests
import json

app = FastAPI()


class Location(BaseModel):
    lat: str
    lon: str
    rad: Optional[int] = 500


class Coordinates(BaseModel):
    lat: str
    lon: str


class VibeOut(BaseModel):
    name: str
    website: str
    lat: float
    lon: float


# Make these configurable ENV/ETC
MAX_RAD = 100000000
MIN_RAD = 100


@app.post("/get_vibes/")
async def get_vibes(loc: Location):
    # Get bar data from OpenStreetMap

    # this doesn't do anything since the bbox in get_locations is defined separately
    # make sure to update eventually to use this
    if loc.rad != None and loc.rad <= MAX_RAD and loc.rad >= MIN_RAD:
        radius = loc.rad

    searchCityState = ''
    try:
        searchCityState = get_reverse_geocode(loc.lat, loc.lon)
    except:
        print('nomimatim reverse geocode error')
    print('searchCityState is: ' + str(searchCityState))
    
    try:
        vibes = get_locations(float(loc.lat), float(loc.lon), radius)
    except:
        print("overpass get locations error")
    # print('locations are: ' + str(vibes))
    # Initialize variables
    id_counter = 0
    output = []
    output.append(searchCityState)
    print('vibe nodes are:')
    try:
        for node in vibes.nodes():
            tags = node.tags()
            curr_vibe = {}
            # build json response from metadata
            try:
                curr_vibe['name'] = tags['name']
            except:
                continue

            try:
                curr_vibe['website'] = tags['website']
            except:
                curr_vibe['website'] = None

            try:
                curr_vibe['lon'] = node.lon()
                curr_vibe['lat'] = node.lat()
            except:
                continue
            # try:
            #   # need to figure out where the address is stored in the nodes, in meantime can use lat and lon.
            #   #curr_vibe['address'] = nodes.tags['addr:housenumber'] +' '+nodes.tags['addr:street']
            #   curr_vibe['houseNumber'] = nodes.tags['addr:housenumber']
            #   curr_vibe['street'] = nodes.tags['addr:street']
            #   curr_vibe['city'] = nodes.tags['addr:city']
            #   curr_vibe['state'] = nodes.tags['addr:state']
            #   curr_vibe['zip'] = nodes.tags['addr:postcode']
            # except:
            #     continue

            output.append(curr_vibe)
        for way in vibes.ways():
            print('way is' + '\n' + str(way))
            tags = way.tags()

            curr_vibe = {}
            # build json response from metadata
            try:
                curr_vibe['name'] = tags['name']
            except:
                continue

            try:
                curr_vibe['website'] = tags['website']
            except:
                curr_vibe['website'] = None

            try:
                curr_vibe['lat'] = way.centerLat()
                curr_vibe['lon'] = way.centerLon()
            except:
                continue
            # try:
            #   # need to figure out where the address is stored in the nodes, in meantime can use lat and lon.
            #   #curr_vibe['address'] = nodes.tags['addr:housenumber'] +' '+nodes.tags['addr:street']
            #   curr_vibe['houseNumber'] = nodes.tags['addr:housenumber']
            #   curr_vibe['street'] = nodes.tags['addr:street']
            #   curr_vibe['city'] = nodes.tags['addr:city']
            #   curr_vibe['state'] = nodes.tags['addr:state']
            #   curr_vibe['zip'] = nodes.tags['addr:postcode']
            # except:
            #     continue

            output.append(curr_vibe)
    except:
        print("error in result")
    json_out = jsonable_encoder(output)
    return JSONResponse(content=json_out)


@app.post("/reverse_geocode/")
async def reverse_geocode(coords: Coordinates):
    # Perform reverse geo-code
    reverse_geo_code_res = ''
    try:
        reverse_geo_code_res = get_reverse_geocode(coords.lat, coords.lon)
    except KeyError:
        print('vibecheck-locations-api: handling exception, get_reverse_geocode could not complete')

    print('reverse_geo_code_res is: ' + str(reverse_geo_code_res))

    output = []
    output.append(reverse_geo_code_res)
    json_out = jsonable_encoder(output)
    return JSONResponse(content=json_out)

# sessionmaker = FastAPISessionMaker(database_uri)
#  Timed Event
# @app.on_event("startup")
# @repeat_every(seconds=60*60*2)
# async def timed_function() -> None:
