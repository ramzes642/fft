package main
import (
    "github.com/jvlmdr/go-fftw/fftw"
    "os"
    "log"
    "encoding/binary"
    "math"
    "fmt"
    "image"
    "image/png"
    "image/color"
    "bytes"
    "bufio"
    "strings"
    "strconv"
    "regexp"
)



var windowSize int;
var graphSize int;
var sampleRate int;
var exportRaw bool;
var exportTone bool;
var exportPng bool;
var windowHann bool;
var img *image.RGBA;
var column []float64;
var binmin float64;

var tones [][]float64;
var toneBase = []float64{
    16.35,
    17.32,
    18.35,
    19.45,
    20.6,
    21.83,
    23.12,
    24.50,
    25.96,
    27.5,
    29.14,
    30.87,
}


func read_int32(data []byte) (ret int16) {
    buf := bytes.NewBuffer(data)
    binary.Read(buf, binary.BigEndian, &ret)
    return
}

func hanning(n int, wavdata []byte) complex128 {
    snd := int16(wavdata[0]) | int16(wavdata[1])<<8;

    var r float64 = float64(snd);

    //log.Printf("%v %v %v", r, snd, wavdata);

    var win float64 = 1;
    if windowHann {
        win = 0.5 * (1 - math.Cos(2 * math.Pi * float64(n) / (float64(windowSize) - 1)));
    }

    return complex(r * win, 0);
}

var tsv *os.File;

func decode(filename string) {
    var file *os.File;
    file, err := os.Open(filename) // For read access.
    if err != nil || file == nil {
        log.Printf("\nFile open error %s\n", err.Error())
        return
    }
    column = make([]float64, windowSize);
    for i := 1; i < windowSize; i++ {
        column[i] = 0;
    }

    var buffer []byte = make([]byte, 48);

    file.Read(buffer);
    if string(buffer[0:4]) != "RIFF" {
        log.Printf("\nNot wave file %s\n", err.Error())
        return
    }

    if windowHann {
        log.Printf("Hanning ON\n")
    } else {
        log.Printf("Hanning OFF\n")
    }
    // wave data size
    subchunkSizeOffset := binary.LittleEndian.Uint32(buffer[16:20]) + 24
    size := binary.LittleEndian.Uint32(buffer[subchunkSizeOffset:subchunkSizeOffset+4]) - 48

    if exportPng {
        img = image.NewRGBA(image.Rect(0, 0, int(size)/graphSize/windowSize, windowSize))
        defer (func() {
            f, err := os.Create(filename+".png");
            if err == nil {
                log.Printf("File writed");

                err := png.Encode(f, img);
                if err != nil {
                    panic(err)
                }
            } else {
                log.Printf("File err");
            }
        })()
    }

    buffer = make([]byte, size);
    file.Read(buffer)

    win := fftw.NewArray(windowSize);
    out := fftw.NewArray(windowSize);
    p := fftw.NewPlan(win, out, fftw.Forward, fftw.Measure)
    defer p.Destroy()

    binmin = float64(math.Inf(1));


    tsv, err = os.Create(filename+".tsv");
    if err != nil {
        log.Fatal("tsv open err");
    } else {
        defer tsv.Close();
    }

    if exportTone {
        tones = make([][]float64, 9);
        for i := range tones {
            tones[i] = make([]float64, 12);
        }
    }

    for i := 0; i < int(size) - windowSize * 2; i += windowSize * 2 {
        // переводим дискретный сигнал в комплексный с наложением окна ханна
        for j := 0; j < windowSize; j++ {
            win.Set(j, hanning(j, buffer[i+j*2 : i+j*2+2]))
        }
        // преобразование
        p.Execute()
        export(out, i)
    }

    if exportTone {

        final_tones := make(map[int]float64, 12);

        // header
        freq_base := toneBase[0]
        fmt.Printf("Freq:")
        for i := 1; i < 9; i++ {
            freq := freq_base * math.Pow(2, float64(i))
            fmt.Printf("\t%.2f", freq)

        }
        fmt.Printf("\n")


        for fbid,_ := range toneBase {
            var note float64 = 0;
            fmt.Printf("%d",fbid)
            for i := 0; i < 9; i++ {
                fmt.Printf("\t%.2f\t", tones[i][fbid])
                note += tones[i][fbid];
            }
            fmt.Printf("\n")
            final_tones[fbid] = note / 9;
        }
        //final_tones.Sort();
        fmt.Printf("TONE: ");




        for i :=0; i < 12; i++ {
            max := 0.0;
            maxidx := 0;
            for idx, val := range final_tones {
                max = math.Max(max, val)
                if val == max {
                    maxidx = idx
                }
            }
            delete(final_tones, maxidx)
            fmt.Printf("%x", maxidx)
        }
        fmt.Printf("\n");

    }

}

func ParseComplex(str string) complex128 {
    //-161822.35142991663+1.1927486116049597e+06i
    reg, err := regexp.Compile("^(.*[^e])([+-].*)i$")

    parts := reg.FindStringSubmatch(str)

    if err != nil {
        return complex(0, 0);
    }

    re, err := strconv.ParseFloat(parts[1], 64);
    if err != nil {
        re = 0;
    }
    im, err := strconv.ParseFloat( parts[2], 64)
    if err != nil || im == 0 {
        im = 0;
    }

    return complex(re, im)
}

func encode(filename string) {
    var file *os.File;
    file, err := os.Open(filename) // For read access.
    if err != nil || file == nil {
        log.Printf("\nFile open error %s\n", err.Error())
        return
    }
    defer file.Close();

    win := fftw.NewArray(windowSize);
    out := fftw.NewArray(windowSize);
    p := fftw.NewPlan(win, out, fftw.Backward, fftw.Measure)
    defer p.Destroy()


    f, err := os.Create(filename+".rev.raw");

    if err == nil {
        log.Printf("File writed");
    }

    defer f.Close()


    scanner := bufio.NewScanner(file)
    //scanner.Buffer(buf,32767);


    logline := true;
    log.Print("new scan");
    for scanner.Scan() {

        data := strings.Split(scanner.Text(),"\t")

        for i := 0; i <= windowSize / 2; i++ {
            val := ParseComplex(data[i]);

            win.Set(i, val )
            if i > 0 {
                win.Set(windowSize - i, complex(real(val), -imag(val)))
            }
        }
        if !logline {
            for i := 0; i < windowSize; i++ {
                log.Printf("%v", win.At(i));
            }
            logline = true;
        }
                //break;
        p.Execute();
        for i := 0; i < windowSize; i++ {
            //log.Printf("%v", win.At(i));
            sample := int16(real(out.At(i)) / float64(windowSize))
            //log.Printf("%v", sample );

          //  snd := byte(sample) | int16(wavdata[1])<<8;

            binary.Write(f, binary.LittleEndian, sample )
        }



       // break;
    }
}

func toneWindow(base_freq float64, actual_freq float64, step_size float64) float64 {
    return 1;
}

func export(win *fftw.Array, pos int) {

    for i := 0; i < windowSize; i++ {
        freq := float64(i) * float64(sampleRate) / float64(windowSize);

        c := win.At(i);
        r := real(c)
        im := imag(c)
        a := math.Sqrt(r*r + im*im);
        column[i] += r*r + im*im;

        mag := a;

        if exportRaw {
            if i > windowSize / 2 {
                break
            }
            tsv.WriteString(strings.Trim(fmt.Sprintf("%v",c),"()")+"\t");
        } else {
            if math.Abs(freq - 500) < 80 {
                fmt.Printf("%0.2f: %0.2f; ", freq, mag)
            }
        }
    }

    if exportTone {
        var fft_step float64 = float64(windowSize) / float64(sampleRate)
        for fbid, freq_base := range toneBase {
            for j := 0; j < 9; j++ {
                freq := freq_base * math.Pow(2, float64(j))
                var span float64;
                if fbid == 0 {
                    span = toneBase[fbid+1] / 2
                } else {
                    span = toneBase[fbid-1] / 2
                }
                win_count := 0;
                for f_idx := int((freq - span)*fft_step); f_idx < int((freq + span) * fft_step); f_idx++ {
                    tones[j][fbid] += math.Abs(real(win.At(f_idx))) * toneWindow(freq, float64(f_idx) * fft_step, span); // sum
                    win_count++;
                }

                tones[j][fbid] /= float64(win_count); // round
            }
        }
    }

    if exportPng && (pos % graphSize == 0) {
        bins := make([]float64, windowSize);
        binmax := float64(0);

        for i := 1; i < windowSize; i++ {
            if column[i] == 0 {
                continue;
            }
            bins[i] = 10 * math.Log10(column[i]);
            column[i] = 0;
            binmax = math.Max(bins[i], binmax)
            binmin = math.Min(bins[i], binmin)
        }

        for i := 0; i < windowSize; i++ {
            cval := float64((bins[i]-binmin) / (binmax-binmin));

            img.SetRGBA(pos/graphSize/windowSize, i, color.RGBA{ pallete_gs(cval), pallete_gs(cval), pallete_gs(cval), 255 })

            //log.Printf("%v %v %v", imin, imax, c);
        }

    }
    tsv.WriteString("\n")
}


/*
0     0   0   0
25    0   0   255
50    255 0   0
75    255 255 0
100   255 255 255
*/

func pallete_gs(v float64) uint8 {

        return uint8(v*255)

}


