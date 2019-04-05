# telecom

> Telecom is WIP and subject to lots of change. This document doesn't accurately represnt the current featureset of telecom during its development.

Telecom is a small Go library which implements the [Discord voice](https://discordapp.com/developers/docs/topics/voice-connections) protocol. Because of the way this protocol was designed telecom can live in an entirely isolated world and still be utilized by any common Discord library. Telecom shares some ideas with the [lavalink](https://github.com/Frederikam/Lavalink) project but has a focus on embedding and performance.

## Supported Interfaces

- **Shared Library** Telecom can be built as a shared library and integrated into any modern language that supports a standard C FFI interface. This provides a great opportunity for libraries to include telecom instead of reinventing the wheel of voice connections.
- **Python Library** Telecom includes a Python library which generates a low-level Python C module and wraps it in a higher level pure-Python library.
- **HTTP Server** Telecom also provides a binary that can be used to run an HTTP server with a simple JSON API. This can be utilized to distribute voice connections across multiple servers for large bots.

## Language Wrappers

- Python 2.7+/3.6+

## Example

### C FFI

```c
client* client = create_client(user_id, guild_id, session_id)

client_set_server_info(client, endpoint, token)

playable* playable = create_playable_from_url("https://discordapp.com/tunes.mp3")
client_play(client, playable)

playable_pause(playable)
playable_stop(playable)

client_set_volume(client, 70)

playable* pipe = create_playable_from_pcm_pipe()
playable_pipe_send(pipe, my_pcm_data)

client_destroy(client)
```

### Python

```python
from telecom import Client, Playable, Output

client = Client(user_id, guild_id, session_id)

# Later when you've joined a voice channel and have received the `endpoint` and `token`...
client.set_server_info(endpoint, token)

# Create a playable and play it
song = Playable.from_url('https://discordapp.com/tunes.mp3')
client.play(song)

# Pause or stop the playable
song.pause()
song.stop()

# Adjust the volume of the client
client.set_volume(70)

# Perhaps we want to stream in some audio data from Python land
pipe = Playable.from_pcm_pipe()
pipe.send(my_pcm_frame)

# Save a single users audio data to a file
user = client.get_user(other_user_id)
output = Output.to_file('bob.mp3')
user.set_output(output)

# Mux everyones audio into a single stream and save it to a file
other_output = Output.to_file('everyone.mp3')
output_muxer = Output.muxer(include_self=False)
output_muxer.set_output(other_output)
client.set_output_muxer(output_muxer)
```
