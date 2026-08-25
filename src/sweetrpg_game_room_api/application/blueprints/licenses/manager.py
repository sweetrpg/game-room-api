# -*- coding: utf-8 -*-
__author__ = "Paul Schifferer <dm@sweetrpg.com>"
"""Resource definitions for licenses.
"""

from flask_rest_jsonapi import ResourceList, ResourceDetail, ResourceRelationship
from sweetrpg_game_room_objects.api.license.schema import LicenseAPISchema
from sweetrpg_api_core.data import APIData
from sweetrpg_game_room_objects.model.license import License
from sweetrpg_game_room_api.application.db import db
from sweetrpg_game_room_api.application.blueprints.setup import model_info


class LicenseList(ResourceList):
    disable_oauth = True
    schema = LicenseAPISchema
    data_layer = {"class": APIData, "type": "license", "model": License, "db": db, "model_info": model_info}


class LicenseDetail(ResourceDetail):
    disable_oauth = True
    schema = LicenseAPISchema
    data_layer = {
        "class": APIData,
        "type": "license",
        "model": License,
        "db": db,
        "model_info": model_info
    }


class LicenseVolumeRelationship(ResourceRelationship):
    schema = LicenseAPISchema
    data_layer = {
        "class": APIData,
        "type": "license",
        "model": License,
        "db": db,
        "model_info": model_info
    }
