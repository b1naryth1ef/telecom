from disco.bot import Plugin
from disco.gateway.packets import OPCode
from telecom import TelecomClient


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

    @Plugin.listen('VoiceServerUpdate')
    def on_voice_server_update(self, event):
        self.vc = TelecomClient(self.state.me.id, event.guild_id, self.client.gw.session_id)
        self.vc.set_server_info(event.endpoint, event.token)

    @Plugin.command('play')
    def on_play(self, event):
        self.vc.play_from_file('yeet.mp3')
