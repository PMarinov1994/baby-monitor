cmake_minimum_required(VERSION 3.16)
project(RpiCameraWrapper)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# Find required packages
find_package(PkgConfig REQUIRED)

# Find libcamera
pkg_check_modules(LIBCAMERA REQUIRED libcamera)
if(NOT LIBCAMERA_FOUND)
    message(FATAL_ERROR "libcamera not found")
endif()

# Find Boost with only the program_options module
find_package(Boost REQUIRED COMPONENTS program_options)

# Check if Boost was found
if(NOT Boost_FOUND)
    message(FATAL_ERROR "Boost program_options not found")
endif()

pkg_check_modules(RPICAM_APP REQUIRED rpicam_app)
if(NOT RPICAM_APP_FOUND)
    message(FATAL_ERROR "rpicam_app not found")
endif()

# Define your source files
set(SOURCES
    rpicam_api.cpp
)

# Create the shared library in the current directory
add_library(${PROJECT_NAME} SHARED ${SOURCES})

# Link dependencies
target_link_libraries(${PROJECT_NAME}
    PRIVATE
        ${LIBCAMERA_LIBRARIES}
        Boost::program_options  # Using imported target for better dependency handling
        ${RPICAM_APP_LIBRARIES}
)

# Include directories
target_include_directories(${PROJECT_NAME}
    PRIVATE
        ${LIBCAMERA_INCLUDE_DIRS}
        ${Boost_INCLUDE_DIRS}
        ${RPICAM_APP_INCLUDE_DIRS}
	"/usr/local/include/rpicam-apps" # Hack because we need it and pkg-config does not provide it
)

# Add compile flags from pkg-config
target_compile_options(${PROJECT_NAME}
    PRIVATE
        ${LIBCAMERA_CFLAGS_OTHER}
        ${RPICAM_APP_CFLAGS_OTHER}
)
