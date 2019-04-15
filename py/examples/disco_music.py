from disco.bot import Plugin
from disco.gateway.packets import OPCode
from telecom import TelecomConnection, AvConvPlayable


class TelecomPlugin(Plugin):
    @Plugin.command('join')
    def on_join(self, event):
        channel = next((i for i in event.guild.channels.values() if i.is_voice), None)
        if not channel:
            return event.msg.reply('no voice channels')

        self.client.gw.send(OPCode.VOICE_STATE_UPDATE, {
            'self_mute': False,
            'self_deaf': False,
            'self_video': False,
            'guild_id': event.guild.id,
            'channel_id': channel.id,
        })

        self.vc = TelecomConnection(self.state.me.id, channel.guild_id, self.client.gw.session_id)

    @Plugin.listen('VoiceServerUpdate')
    def on_voice_server_update(self, event):
        print('Forwarding VoiceServerUpdate')
        self.vc.set_server_info(event.endpoint, event.token)

    @Plugin.command('play', '<path:str>')
    def on_play(self, event, path):
        playable = AvConvPlayable(path)
        self.vc.play(playable)

    @Plugin.command('leave')
    def on_leave(self, event):
        if hasattr(self, 'vc'):
            del self.vc

            self.client.gw.send(OPCode.VOICE_STATE_UPDATE, {
                'self_mute': False,
                'self_deaf': False,
                'self_video': False,
                'guild_id': event.guild.id,
                'channel_id': None,
            })
