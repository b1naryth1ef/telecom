import setuptools
from distutils.core import setup, Extension

telecom_module = Extension(
    'telecom.telecom',
    include_dirs=['../cmd/telecom-native/'],
    libraries=['telecom'],
    library_dirs=['../cmd/telecom-native/'],
    sources=['telecom.c'],
)

setuptools.setup(
    name='telecom-py',
    version='0.0.1',
    author='b1nzy',
    author_email='b1naryth1ef@gmail.com',
    description='Discord voice client',
    url='https://github.com/b1naryth1ef/telecom',
    packages=['telecom'],
    ext_modules=[telecom_module],
    classifiers=[
        'Topic :: Internet',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
        'Topic :: Utilities',
    ],
)
