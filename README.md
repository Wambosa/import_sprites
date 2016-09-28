# import_sprites
sprite importer that extracts an extreme amount of meta data from the file name in order to manage complex inGame events and animations

## What it does
There are nearly 2000+ image file art assets created by team artist(s).
There are 4 types of images, each with differing meta data (character, decoration, animation, and map). 
These image files have relationships with one another. The job of this script is to discover and record those relationships in a relational database so that the game can access the correct information programatically in a manageable way.


```axeatkdown001.png``` will have ```SpriteSlice``` record relating to a ```Sprite``` record.

The folder structure determines the parent sprite object name, whilst the .png files are individual sprite frames (or slices) 

an example ```one to many``` recordset:

**Sprite**

sprite_id | sprite_name | type | image_count | tiled_width | tiled_height | pixels | direction_support
--------- | ----------- | ---- | ----------- | ----------- | ------------ | ------ | -----------------
28 | mara | actor | 244 | 1 | 1 | 128 | 4

**Sprite Slice**

slice_id | sprite_id | frame_number | direction | sprite_action_id | frame_seconds | event_id | unity_path
-------- | --------- | ------------ | --------- | ---------------- | ------------- | -------- | ----------
826 | 28 | 1 | down | 17 | 0.08 | 65 | "sprites/axeatkdown001.png"
827 | 28 | 2 | down | 17 | 0.08 | 65 | "sprites/axeatkdown002.png"
827 | 28 | 3 | down | 17 | 0.08 | 65 | "sprites/axeatkdown003.png"

### notes
- stashing this script for my code journal
- designed for a specific 2d game made nearly 5 years ago
- this script is **not** highly configurable. For example; only two folder levels are walked due to the specific file structure of _Regal Arbor_. Additionally the file paths are hardcoded forcing a recompile if build directory is ever changed.
