# -*- coding: utf-8 -*-
__author__ = "Paul Schifferer <dm@sweetrpg.com>"
"""
"""

from flask_rest_jsonapi import ResourceList, ResourceDetail, ResourceRelationship
from  sweetrpg_game_room_objects.api.system.schema import SystemAPISchema
from sweetrpg_api_core.data import APIData
from sweetrpg_game_room_objects.model.system import System
from sweetrpg_game_room_api.application.db import db
from sweetrpg_game_room_api.application.blueprints.setup import model_info


class SystemList(ResourceList):
    schema = SystemAPISchema
    data_layer = {"class": APIData, "type": "system", "model": System, "db": db, "model_info": model_info}


class SystemDetail(ResourceDetail):
    schema = SystemAPISchema
    data_layer = {
        "class": APIData,
        "type": "system",
        "model": System,
        "db": db,
        "model_info": model_info
    }


# class SystemAuthorRelationship(ResourceRelationship):
#     schema = SystemAPISchema
#     data_layer = {
#         "class": APIData,
#         "type": "system",
#         "model": System,
#         "db": db,
#         "model_info": model_info
#     }
