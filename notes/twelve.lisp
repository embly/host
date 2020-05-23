; Ronin v2.50

(def colors
  ("#000000" "#72dec2"))

;
(clear)

(def size 300)


; circle
(stroke
  (circle size size (div size 2)) colors:1)


: arc with circle's diameter as radius
(stroke
  (arc 450 size size
    (rad 120)
    (rad -120)) colors:1)

(stroke
  (arc 150 size size
    (rad -60)
    (rad 60)) colors:1)



; guide lines
(stroke
  (line size 0 size (mul size 2)) colors:1)

(stroke
  (line 0 size (mul size 2) size) colors:1)


; arc's with circle's radius drawn from the
; intersection of the initial arcs and the
; guide lines

(stroke
  (arc 150 size (div size 2)
    (rad -60)
    (rad 60)) colors:0)

(stroke
  (arc 450 size (div size 2)
    (rad 120)
    (rad -120)) colors:0)


(def south (arc size 450 (div size 2)
    (rad -150) (rad -30)))

(stroke
   south colors:0)

(def north (arc size 150 (div size 2)
    (rad 30) (rad 150)))

(stroke
  north
   colors:0)
