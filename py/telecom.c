#define Py_LIMITED_API
#include <telecom.h>
#include <Python.h>

static void client_destructor(PyObject* self) {
  telecom_client_destroy((GoInt)PyCapsule_GetPointer(self, NULL));
}

static void playable_destructor(PyObject* self) {
  telecom_playable_destroy((GoInt)PyCapsule_GetPointer(self, NULL));
}

static PyObject* create_client(PyObject* self, PyObject* args) {
  char* user_id;
  char* guild_id;
  char* session_id;

  if (!PyArg_ParseTuple(args, "sss", &user_id, &guild_id, &session_id)) {
    return NULL;
  }

  GoInt handle = telecom_create_client(user_id, guild_id, session_id);
  return PyCapsule_New((void*)handle, NULL, &client_destructor);
}

static PyObject* client_update_server_info(PyObject* self, PyObject* args) {
  PyObject* client;
  char* endpoint;
  char* token;

  if (!PyArg_ParseTuple(args, "Oss", &client, &endpoint, &token)) {
    return NULL;
  }

  GoInt handle = (GoInt)PyCapsule_GetPointer(client, NULL);
  telecom_client_update_server_info(handle, endpoint, token);
  return Py_BuildValue("");
}

static PyObject* client_play(PyObject* self, PyObject* args) {
  PyObject* client;
  PyObject* playable;

  if (!PyArg_ParseTuple(args, "OO", &client, &playable)) {
    return NULL;
  }

  GoInt clientHandle = (GoInt)PyCapsule_GetPointer(client, NULL);
  GoInt playableHandle = (GoInt)PyCapsule_GetPointer(playable, NULL);
  telecom_client_play(clientHandle, playableHandle);
  return Py_BuildValue("");
}

static PyObject* create_avconv_playable(PyObject* self, PyObject* args) {
  char* path;

  if (!PyArg_ParseTuple(args, "s", &path)) {
    return NULL;
  }

  GoInt handle = telecom_create_avconv_playable(path);
  return PyCapsule_New((void*)handle, NULL, &playable_destructor);
}

static PyMethodDef TelecomMethods[] = {
  {"create_client", create_client, METH_VARARGS, "Create a new telecom client."},
  {"client_update_server_info", client_update_server_info, METH_VARARGS, "Set telecom client server info."},
  {"client_play", client_play, METH_VARARGS, "Play a playable."},
  {"create_avconv_playable", create_avconv_playable, METH_VARARGS, "Create a playable."},
  {NULL, NULL, 0, NULL}
};

#if PY_MAJOR_VERSION >= 3
static struct PyModuleDef telecommodule = {
  PyModuleDef_HEAD_INIT, "telecom", NULL, -1, TelecomMethods
};

PyMODINIT_FUNC PyInit_telecom() {
  telecom_setup_logging(1, 1);
  return PyModule_Create(&telecommodule);
}
#else
PyMODINIT_FUNC inittelecom(void) {
  telecom_setup_logging(1, 1);
  Py_InitModule("telecom", TelecomMethods);
}
#endif
