package main

import (
	"os"
	"fmt"
	"image"
	"regexp"
	"errors"
	"strconv"
	"io/ioutil"
	_ "image/png"
	"database/sql"
	"github.com/wambosa/easydb"
)

var (
	ConnectionString = "c:/work/git/regal_arbor/ra.db3"
	MetadataSearchMap = map[string]SpriteSliceMetadata{}
)

type ImageFile struct {
	name string
	parent string
	sType string
	width int
	height int
	isFractal bool

	spriteActionId int
	spriteActionName string

	direction string
	frameNumber int
}

type Sprite struct {
	id int
	name string
	sType string
	imageCount int
	tiledWidth int
	tiledHeight int
	pixels int
	directionSupport int

	hasUp bool
	hasRight bool
	hasDown bool
	hasLeft bool
}

type SpriteSlice struct {
	spriteId int
	frameNumber int
	direction string
	spriteActionId int
	frameSeconds float64
	eventId int
	eventJson string
	unityPath string
}

type SpriteSliceMetadata struct {
	matchText string
	minFrame int
	maxFrame int
	frameSeconds float64
	eventId int
	eventJson string
}

func main() {

	fmt.Println("Regal Arbor Sprite Import v3")

	allFiles, err := GetImageFiles("C:/work/git/regal_arbor/assets/resources/sprites/")

	if err != nil {fatal("problem loading sprite files", err)}

	allFiles = ExtractEncodedData(allFiles)

	sprites := GetSpritesFromImageFiles(allFiles)

	fmt.Println("Will insert ", len(sprites), " new sprites")

	InsertNewSprites(sprites)

	MetadataSearchMap = GetMetadataSearchMap()

	spriteSlices, err := CreateSpriteSlices(allFiles)

	if err != nil {fatal("problem creating sprite slices", err)}

	fmt.Println("Will insert ", len(spriteSlices), " new sprite_slices")

	InsertNewSpriteSlices(spriteSlices)

	fmt.Println("done!")
}

func GetImageFiles(path string)([]ImageFile, error) {

	spriteTypeFolders := []string {
		"maps",
		"houses",
		"zepps",
		"characters",
		"decorations",
	}

	allFiles := make([]ImageFile, 0)

	for _, spriteType := range spriteTypeFolders {

		files, err := ioutil.ReadDir(path + spriteType)

		if err != nil {return allFiles, err}

		// there will only ever be max two levels deep concerning sprite folders ever.
		for _, file := range files {

			if (file.IsDir()) {

				subFiles, err := ioutil.ReadDir(path+spriteType+"/"+file.Name())

				if err != nil {return allFiles, err}

				for _, subFile := range subFiles {

					if (subFile.IsDir()) {continue}
					if (!IsSupportedFile(subFile)){continue}

					width, height, err := GetImageDimension(path+spriteType+"/"+file.Name()+"/"+subFile.Name())

					if err != nil {return allFiles, errors.New(subFile.Name()+"\n"+err.Error())}

					allFiles = append(allFiles, ImageFile{
						name: subFile.Name(),
						parent: file.Name(), // this is is the subfolder name
						sType: spriteType,
						width: width,
						height: height,
						isFractal: true,
					})
				}

			}else {

				if (!IsSupportedFile(file)){continue}

				width, height, err := GetImageDimension(path+spriteType+"/"+file.Name())

				if err != nil {return allFiles, errors.New(file.Name()+"\n"+err.Error())}

				allFiles = append(allFiles, ImageFile{
					name: file.Name(),
					parent: spriteType,
					sType: spriteType,
					width: width,
					height: height,
					isFractal: false,
				})
			}
		}
	}
		return allFiles, nil
}

func GetSpritesFromImageFiles(imageFiles []ImageFile)([]Sprite){

	tiledDimensions, err := easydb.RunQuery("sqlite3", ConnectionString,
		"SELECT * FROM sprite_megatile_size")

	if err != nil {fatal("unable to load dimensions", err)}

	sprites := map[string]Sprite{}

	for _, imageFile := range imageFiles {

		uniqueName := imageFile.name[:len(imageFile.name)-4]

		if imageFile.isFractal {
			uniqueName = imageFile.parent}

		if _,exists := sprites[uniqueName]; exists {

			sprites[uniqueName] = UpdateSpriteDirectionAndCount(sprites[uniqueName], imageFile)

		}else{

			tiledWidth, tiledHeight := GetTiledDimensions(uniqueName, tiledDimensions)

			sprites[uniqueName] = Sprite {
				name: uniqueName,
				sType: imageFile.sType,
				imageCount: 1,
				tiledWidth: tiledWidth,
				tiledHeight: tiledHeight,
				pixels: imageFile.width, // assumes that the creator made the image square
				directionSupport: 0,
			}
		}
	}

	spriteArray := make([]Sprite, len(sprites))

	i := 0
	for _, sprite := range sprites {
		spriteArray[i] = sprite
		i++
	}

	return spriteArray
}

func InsertNewSprites(sprites []Sprite)(error) {
	//todo: need to capture failures in case of column misspelling.
	query := `
	INSERT INTO sprite
	(sprite_name, type, image_count, tiled_width, tiled_height, pixels, direction_support)
	Values(?, ?, ?, ?, ?, ?, ?)
	`

	db, err := sql.Open("sqlite3", ConnectionString)

	if(err != nil){return err}

	defer db.Close()

	for _, sprite := range sprites {

		_, err := easydb.Exec(db, query,
			sprite.name,
			sprite.sType,
			sprite.imageCount,
			sprite.tiledWidth,
			sprite.tiledHeight,
			sprite.pixels,
			sprite.directionSupport)

		if(err != nil){return err}
	}

	return nil
}

func InsertNewSpriteSlices(spriteSlices []SpriteSlice)(error) {

	query := `
	INSERT INTO sprite_slice
	(sprite_id,
	frame_number,
	direction,
	sprite_action_id,
	frame_seconds,
	event_id,
	event_json,
	unity_path)
	Values(?,?,?,?,?,?,?,?)
	`

	db, err := sql.Open("sqlite3", ConnectionString)

	if(err != nil){return err}

	defer db.Close()

	for _, sSlice := range spriteSlices {

		_, err := easydb.Exec(db, query,
			sSlice.spriteId,
			sSlice.frameNumber,
			sSlice.direction,
			sSlice.spriteActionId,
			sSlice.frameSeconds,
			sSlice.eventId,
			sSlice.eventJson,
			sSlice.unityPath)

		if(err != nil){return err}
	}

	return nil
}

func CreateSpriteSlices(imageFiles []ImageFile)([]SpriteSlice, error) {

	spritePieces := make([]SpriteSlice, len(imageFiles))

	dbSprites, err := easydb.RunQuery("sqlite3", ConnectionString,
		"SELECT sprite_id, sprite_name FROM sprite")

	if err != nil {return spritePieces, err}
	if len(dbSprites) == 0 {return spritePieces, errors.New("The Sprites have not been saved to the database yet")}
	officialSpriteIds := map[string]int{}

	//this relies on the data having duplicates (each sprite name must be unique or else there is a high change things will not match up as intended)
	for _, sprite := range dbSprites {

		spriteName := string(sprite["sprite_name"].([]uint8))

		if _,exists := officialSpriteIds[spriteName]; !exists {
			officialSpriteIds[spriteName] = int(sprite["sprite_id"].(int64))}
	}

	for i, file := range imageFiles {

		extensionlessName := file.name[:len(file.name)-4]

		parentName, unityPath := extensionlessName, extensionlessName

		if file.isFractal {
			parentName = file.parent
			unityPath = file.parent+"/"+extensionlessName
		}

		tempMeta := FindMetadataForThisImageFile(unityPath, file.frameNumber)

		spritePieces[i] = SpriteSlice {
			spriteId: officialSpriteIds[parentName],
			frameNumber: file.frameNumber,
			direction: file.direction,
			spriteActionId: file.spriteActionId,
			frameSeconds: tempMeta.frameSeconds,
			eventId: tempMeta.eventId,
			eventJson: tempMeta.eventJson,
			unityPath: unityPath,
		}
	}

	return spritePieces, nil
}

func ExtractEncodedData(imageFiles []ImageFile)([]ImageFile) {

	spriteActions, err := easydb.RunQuery("sqlite3", ConnectionString,
		"SELECT * FROM sprite_action")

	if err != nil {fatal("issue getting sprite actions", err)}

	for i, file := range imageFiles {

		imageFiles[i].spriteActionId, imageFiles[i].spriteActionName = GetSpriteActionIdAndName(file, spriteActions)

		imageFiles[i].direction = GetSpriteDirection(file)

		imageFiles[i].frameNumber = GetSpriteFrameNumber(file)
	}

	return imageFiles
}

func GetImageDimension(imagePath string) (int, int, error) {

	file, err := os.Open(imagePath)

	defer file.Close()

	if err != nil { return 0, 0, err}

	image, _, err := image.DecodeConfig(file)

	if err != nil { return 0, 0, err}

	return image.Width, image.Height, nil
}

func IsSupportedFile(aFile os.FileInfo)(bool){

	acceptableFileExtensions := []string {
		"png",
	}

	for _, ext := range acceptableFileExtensions {
		if match,_ := regexp.MatchString(`\.`+ext+`\z`, aFile.Name()); match{
			return true}
	}

	return false
}

func GetMetadataSearchMap()(map[string]SpriteSliceMetadata) {

	spriteMetas, err := easydb.RunQuery("sqlite3", ConnectionString,
		"SELECT * FROM sprite_slice_meta")

	if err != nil {fatal("issue getting sprite slice meta", err)}


	metadataSearchMap := map[string]SpriteSliceMetadata{}

	for _, meta := range spriteMetas {

		matchText := string(meta["match_text"].([]uint8))

		metadataSearchMap[matchText] = SpriteSliceMetadata {
			matchText: matchText,
			minFrame: int(meta["start_frame"].(int64)),
			maxFrame: int(meta["end_frame"].(int64)),
			frameSeconds: meta["frame_seconds"].(float64),
			eventId: int(meta["event_id"].(int64)),
			eventJson: string(meta["event_json"].([]uint8)),
		}
	}

	return metadataSearchMap
}

func FindMetadataForThisImageFile(unityPath string, frameNumber int)(SpriteSliceMetadata){

	returnMetadata := SpriteSliceMetadata {
		frameSeconds: 0.08,
	}

	// the last valid match wins.
	for key, meta := range MetadataSearchMap {

		if isIn,_ := regexp.MatchString(key, unityPath); isIn {

			isAllFrames := (meta.minFrame - meta.maxFrame == 0)
			isInFrameBounds := (frameNumber >= meta.minFrame && frameNumber <= meta.maxFrame)

			if(isAllFrames || isInFrameBounds) {
				returnMetadata = SpriteSliceMetadata{
					eventId: meta.eventId,
					eventJson: meta.eventJson,
					frameSeconds: meta.frameSeconds,
				}
			}
		}
	}

	return returnMetadata
}

func GetSpriteActionIdAndName(spriteFile ImageFile, spriteActions []map[string]interface{})(int, string){

	for _, spriteAction := range spriteActions {

		actionName := string(spriteAction["sprite_action_name"].([]uint8))

		if match,_ := regexp.MatchString(actionName, spriteFile.name); match {
			return int(spriteAction["sprite_action_id"].(int64)), actionName}
	}

	return 0, "MISSING!!!"
}

func GetTiledDimensions(uniqueName string, spriteDimensions []map[string]interface{})(int, int){

	for _, spriteDimension := range spriteDimensions {

		nameToMatch := string(spriteDimension["sprite_name"].([]uint8))

		if match,_ := regexp.MatchString(nameToMatch+`\b`, uniqueName); match{
			return int(spriteDimension["tiled_width"].(int64)), int(spriteDimension["tiled_height"].(int64))}
	}

	return 1, 1
}

func GetSpriteDirection(spriteFile ImageFile)(string) {

	directions := []string {
		"UP",
		"RIGHT",
		"DOWN",
		"LEFT",
	}

	for _, direction := range directions {
		if match,_ := regexp.MatchString(`(?i)`+direction, spriteFile.name); match {
			return direction}
	}

	return "DOWN"
}

func GetSpriteFrameNumber(file ImageFile)(int) {

	match := regexp.MustCompile(`\d{2,3}`).FindString(file.name)

	if  match != "" {
		frame, _ := strconv.ParseInt(match, 10, 64)
		return int(frame)
	}

	return 0
}

func UpdateSpriteDirectionAndCount(sprite Sprite, file ImageFile)(Sprite) {

	switch (file.direction) {
		case "DOWN": sprite.hasDown = true
		case "LEFT": sprite.hasLeft = true
		case "UP": sprite.hasUp = true
		case "RIGHT": sprite.hasRight = true
	}

	directionSupport := 0

	if sprite.hasDown {directionSupport ++}
	if sprite.hasLeft {directionSupport ++}
	if sprite.hasUp {directionSupport ++}
	if sprite.hasRight {directionSupport ++}

	sprite.directionSupport = directionSupport
	sprite.imageCount ++

	return sprite
}

func fatal(customMessage string, err error) {
	fmt.Println("FATAL: ", customMessage)
	fmt.Println(err)
	os.Exit(1)
}