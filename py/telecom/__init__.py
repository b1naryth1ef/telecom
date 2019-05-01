from . import telecom as native


class TelecomException(Exception):
    pass


class TelecomConnection(object):
    def __init__(self, user_id, guild_id, session_id):
        self._handle = native.create_client(str(user_id), str(guild_id), str(session_id))

    def update_server_info(self, endpoint, token):
        native.client_update_server_info(self._handle, endpoint, token)

    def play(self, playable):
        native.client_play(self._handle, playable.handle)


class AvConvPlayable(object):
    def __init__(self, path):
        self._handle = native.create_avconv_playable(path)

    @property
    def handle(self):
        return self._handle
