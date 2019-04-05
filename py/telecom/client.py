from telecom import telecom as native


class TelecomClient(object):
    def __init__(self, user_id, guild_id, session_id):
        self._handle = native.create_client(str(user_id), str(guild_id), str(session_id))

    def set_server_info(self, endpoint, token):
        native.client_set_server_info(self._handle, endpoint, token)

    def play_from_file(self, path):
        native.client_play_from_file(self._handle, path)
