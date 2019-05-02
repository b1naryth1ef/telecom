try:
    from setuptools import setup, Extension
except ImportError:
    from distutils.core import setup, Extension

import shutil
import os
import re
import subprocess
from setuptools.command.build_ext import build_ext
from pkg_resources import parse_version


class HybridGoExtension(Extension):
    def __init__(self, name, go_modules, go_links=None, shared=False, **kwargs):
        Extension.__init__(self, name, **kwargs)
        self.go_modules = go_modules
        self.go_links = go_links or {}
        self.shared = shared
        self.skip_go_build = False


telecom_go_ext = HybridGoExtension(
    'telecom.telecom',
    go_modules={
        'telecom': 'github.com/b1naryth1ef/telecom/cmd/telecom-native',
    },
    go_links={
        'github.com/b1naryth1ef/telecom': os.environ.get('TELECOM_REPO_PATH', '..')
    },
    shared=bool(int(os.environ.get('TELECOM_BUILD_SHARED', '0'))),
    sources=['telecom.c'],
    libraries=['telecom'],
)

# In some situations (build wheels) we may want to build the telecom native libs
#  ourselves and pass them in to be linked against. To support this we allow
#  passing an explicit path to the library files (header + .so or .a) via this
#  env variable, which will entirely disable our Go compilation steps.
libpath = os.environ.get('TELECOM_LIB_PATH')
if libpath:
    telecom_go_ext.include_dirs.append(libpath)
    telecom_go_ext.library_dirs.append(libpath)
    telecom_go_ext.skip_go_build = True


class CustomBuildExt(build_ext):
    def run(self):
        for ext in self.extensions:
            if isinstance(ext, HybridGoExtension) and not ext.skip_go_build:
                self.build_go_hybrid(ext)

        build_ext.run(self)

    def build_go_hybrid(self, ext):
        # We require Go v1.12 or greater, so lets check that here
        version_output = subprocess.check_output(['go', 'version']).decode('ascii')
        match = re.match(r'go version go([\d\.]+)', version_output)
        if not match:
            raise Exception('failed to find go version: {}'.format(version_output))

        version = parse_version(match.group(1))
        if version < parse_version('1.12'):
            raise Exception('go version too low (have {} want {})'.format(
                version,
                '>=1.12'
            ))

        # Setup and stage our build directory
        if not os.path.exists(self.build_temp):
            os.mkdir(self.build_temp)

        # Create a temporary GOPATH for our build process
        temp_gopath = os.path.join(self.build_temp, 'temp_gopath')
        if os.path.exists(temp_gopath):
            shutil.rmtree(temp_gopath)

        os.mkdir(temp_gopath)
        os.mkdir(os.path.join(temp_gopath, 'src'))

        env = os.environ.copy()
        env['GOPATH'] = os.path.abspath(temp_gopath)

        # For some use cases it may be useful that the code within our gopath
        #  does not reflect the latest HEAD of the Go repository, but rather is
        #  linked in from sources our Python package provides. To faciliate this
        #  we link these sources into our gopath now.
        for url, relative_path in ext.go_links.items():
            base, name = os.path.split(url)
            os.makedirs(os.path.join(temp_gopath, 'src', base))

            absolute_url_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), relative_path)
            os.symlink(absolute_url_path, os.path.join(temp_gopath, 'src', url))

        lib_file_ext = '.so' if ext.shared else '.a'
        for name, url in ext.go_modules.items():
            print('[go] building {} library {} from module {}'.format(
                'shared' if ext.shared else 'static',
                name,
                url,
            ))
            artifact_path = os.path.join(self.build_temp, name + lib_file_ext)

            subprocess.check_call(
                ['go', 'get', url],
                env=env,
            )

            subprocess.check_call(
                ['go', 'build', '-o', artifact_path, '-buildmode=c-archive', url],
                env=env,
            )

            # Preface our library with 'lib' as required by most linkers in this situation
            os.rename(
                os.path.join(self.build_temp, name + lib_file_ext),
                os.path.join(self.build_temp, 'lib' + name + lib_file_ext)
            )

        ext.include_dirs.append(self.build_temp)
        ext.library_dirs.append(self.build_temp)


setup(
    name='telecom-py',
    version='0.0.3',
    author='b1nzy',
    author_email='b1naryth1ef@gmail.com',
    description='Discord voice client',
    url='https://github.com/b1naryth1ef/telecom',
    packages=['telecom'],
    ext_modules=[telecom_go_ext],
    cmdclass={
        'build_ext': CustomBuildExt,
    },
    python_requires=">=2.7, !=3.0.*, !=3.1.*, !=3.2.*, !=3.3.*, !=3.4.*",
    classifiers=[
        'Topic :: Internet',
        'Topic :: Software Development :: Libraries',
        'Topic :: Software Development :: Libraries :: Python Modules',
        'Topic :: Utilities',
        'Programming Language :: C',
        'Programming Language :: Python :: 2',
        'Programming Language :: Python :: 2.7',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.5',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
    ],
)
