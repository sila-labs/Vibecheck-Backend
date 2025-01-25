import overpy
from OSMPythonTools.overpass import Overpass, overpassQueryBuilder
import json


def get_locations(latitude, longitude, radius):
    # Initialize the API
    # api = overpy.Overpass()
    api = Overpass(endpoint='http://vibecheck.tech:12346/api/')
    # Define the query
    # query = """(nwr["amenity"~"bar|nightclub|pub|biergarten"](around:{rad},{lat},{lon}););(._;>;);out center;""".format(rad=radius, lat=latitude, lon=longitude)
    lat_south = latitude-0.1
    lon_west = longitude-0.1
    lat_north = latitude+0.1
    lon_east = longitude+0.1
    # query = """(nwr["amenity"~"bar|nightclub|pub|biergarten"]({lat_south},{lon_west},{lat_north},{lon_east}););(._;>;);out center;""".format(lat_south=latitude-0.05, lon_west=longitude-0.05, lat_north=latitude+0.05, lon_east=longitude+0.05)
    query = overpassQueryBuilder(bbox=[lat_south, lon_west, lat_north, lon_east], elementType=['node','way'], selector='"amenity"~"bar|nightclub|pub|biergarten"', out='center')

    # Call the API
    result = api.query(query)
    print('result node count is:')
    print(result.countNodes())
    print('result way count is:')
    print(result.countWays())
#     print(api.parse_json(result, encoding='utf-8'))
    return result
