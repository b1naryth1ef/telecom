from distutils.core import setup, Extension

telecom_module = Extension(
    'telecom.telecom',
    include_dirs=['../cmd/telecom-native/'],
    libraries=['telecom'],
    library_dirs=['../cmd/telecom-native/'],
    sources=['telecom.c'],
)

setup(
    name='telecom',
    version='1.0',
    packages=['telecom'],
    ext_modules=[telecom_module]
)
