#define Py_LIMITED_API
#include <telecom.h>
#include <Python.h>

static void client_destructor(PyObject* self) {
  telecom_client_destroy((GoInt)PyCapsule_GetPointer(self, NULL));
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

static PyObject* client_set_server_info(PyObject* self, PyObject* args) {
  PyObject* client;
  char* endpoint;
  char* token;

  if (!PyArg_ParseTuple(args, "Oss", &client, &endpoint, &token)) {
    return NULL;
  }

  GoInt handle = (GoInt)PyCapsule_GetPointer(client, NULL);
  telecom_client_set_server_info(handle, endpoint, token);
  return Py_BuildValue("");
}

static PyObject* client_play_from_file(PyObject* self, PyObject* args) {
  PyObject* client;
  char* file;

  if (!PyArg_ParseTuple(args, "Os", &client, &file)) {
    return NULL;
  }

  GoInt handle = (GoInt)PyCapsule_GetPointer(client, NULL);
  telecom_client_play_from_file(handle, file);

  return Py_BuildValue("");
}

static PyMethodDef TelecomMethods[] = {
  {"create_client", create_client, METH_VARARGS, "Create a new telecom client."},
  {"client_set_server_info", client_set_server_info, METH_VARARGS, "Set telecom client server info."},
  {"client_play_from_file", client_play_from_file, METH_VARARGS, "Play from a file."},
  {NULL, NULL, 0, NULL}
};

#if PY_MAJOR_VERSION >= 3
static struct PyModuleDef telecommodule = {
  PyModuleDef_HEAD_INIT, "telecom", NULL, -1, TelecomMethods
};

PyMODINIT_FUNC PyInit_telecom() {
  return PyModule_Create(&telecommodule);
}
#else
PyMODINIT_FUNC inittelecom() {
  Py_InitModule("telecom", TelecomMethods);
}
#endif
