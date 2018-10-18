package main

import (
	"fmt"
	"image"
	_"image/jpeg"
	"os"
	"strconv"
	"io/ioutil"
	"github.com/golang/freetype"
	"image/color"
	"strings"
	"bufio"
	"image/png"
	"image/draw"
	"qiniupkg.com/x/log.v7"
	"io"
	"image/gif"
	"path"
	"image/color/palette"
	"time"
	"sync"
	"sort"
)
//动图帧结构
type GifFrame struct {
	index int
	p *image.Paletted
}
var (
	dpi  = 72
	fontfile = "/System/Library/Fonts/Menlo.ttc"
)
var size float64
var pngX ,pngY int
func main() {
	args := os.Args //获取用户输入的所有参数
	if len(args)<2 {
		help()
		return
	}
	t1 := time.Now()
	fmt.Println("...字符画生成中...")
	source := args[1]
	out := args[2]
	sizeArg ,_:=strconv.ParseFloat(args[3],64)
	size = sizeArg

	ImageChange(source,out)
	t2 := time.Now()
	fmt.Println("生成完毕!")
	fmt.Print("耗时:")
	fmt.Println(t2.Sub(t1))
}
var help = func() {
	fmt.Println("参数错误，请按照下面的格式输入：")
	fmt.Println("使用: ./main [输入文件] [输出文件 gif输入输出gif,其他统一png格式] [字符大小 float]")

}

func ImageChange(imgPath , outPath string)  {

	file , err := os.Open(imgPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	ext :=path.Ext(imgPath)
	if ext == ".gif" {
		//判断outPath
		outExt := path.Ext(outPath)
		if outExt !=".gif" {
			fmt.Println("输出图片应该为.gif格式")
			return
		}
		gifChange(file,outPath)
	}else{
		img , _ , err := image.Decode(file)

		if err != nil {
			fmt.Println(err)
			return
		}
		ascllimage(img,outPath)
	}


}
/**
拿到所有帧的图转成 单独的字符画
 */
func gifChange(file io.Reader,out string)  {
	var wg sync.WaitGroup
	var frams []GifFrame
	g , err := gif.DecodeAll(file)
	if err != nil {
		panic(err)
	}
	newGif := gif.GIF{LoopCount:g.LoopCount,Delay:g.Delay}
	ch := make(chan GifFrame,len(g.Image))//创建缓冲通道
	for key,i := range g.Image {
		wg.Add(1)
		go func(m image.Image,index int) {
			p := createPalette(m,"gif"+strconv.Itoa(index)+".png")
			//写入到通道内
			ch <- GifFrame{index:index,p:p}
			defer wg.Done()
		}(i,key)

	}
	wg.Wait()
	close(ch)
	for m:=range ch {
		frams = append(frams,m)
	}
	sort.Slice(frams, func(i, j int) bool {
		return frams[i].index < frams[j].index
	})
	for _,j := range frams {
		newGif.Image = append(newGif.Image,j.p)
	}
	f,_:=os.Create(out)
	defer f.Close()
	gif.EncodeAll(f,&newGif)
}

func createPalette(img image.Image,outPath string) *image.Paletted {
	ascllimage(img,outPath)
	f1 ,_:= os.Open(outPath)
	defer os.Remove(outPath)
	defer f1.Close()
	g1,_ := png.Decode(f1)
	p := image.NewPaletted(image.Rect(0,0,pngX,pngY),palette.Plan9)
	draw.Draw(p,p.Bounds(),g1,image.ZP,draw.Src)
	return p
}
//合成
func drawImg(str ,out string) {
	txt := []byte(str)
	fontBytes, err := ioutil.ReadFile(fontfile)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fg,bg :=image.Black,image.White
	ruler := color.RGBA{0x22,0x22,0x22,0xff}

	my := strings.Split(str, "\n")
	pngY = len(my)* int(size)
	pngX = len([]byte(my[0]))*int(size)
	rgba := image.NewRGBA(image.Rect(0, 0, pngX, pngY))

	draw.Draw(rgba, rgba.Bounds(), bg, image.ZP, draw.Src)

	c := freetype.NewContext()

	c.SetDPI(float64(dpi))
	c.SetFont(f)
	c.SetFontSize(size)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetSrc(fg)

	rgba.Set(0,0,ruler)

	pt := freetype.Pt(1,int(c.PointToFixed(size)>>6))

	for _, s := range txt {
		_, err = c.DrawString(string(s), pt)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		pt.X += c.PointToFixed(size)
		if string(s)=="\n" {
			pt.X = 1
			pt.Y += c.PointToFixed(size)
		}

	}
	outFile, err := os.Create(out)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	b := bufio.NewWriter(outFile)
	err = png.Encode(b, rgba)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	//
	err = b.Flush()
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	fmt.Println("字符画 "+out+" 生成OK.")
}
//图片转为字符画
func ascllimage(m image.Image, out string) {
	var str string
	bounds := m.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()
	arr := []string{"M", "N", "H", "Q", "$", "O", "C", "?", "*", ">", "!", ":", "-", ";", "."}

	for i := 0; i < dy; i++ {
		for j := 0; j < dx; j++ {
			colorRgb := m.At(j, i)
			_, g, _, _ := colorRgb.RGBA()
			avg := uint8(g >> 8)
			num := avg / 18
			str +=arr[num]
			if j == dx-1 {
				str+="\n"
			}
		}
	}
	drawImg(str,out)
}
