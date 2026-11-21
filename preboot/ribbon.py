"""Rasterise the two Silk Bezier contours for firmware without font services."""
curves = [
    [(86,7),(74,26),(51,30),(34,43),(21,54),(27,60),(43,66),(16,64),(5,55),(11,40),(21,23),(57,17),(86,7)],
    [(16,93),(28,74),(51,70),(68,57),(81,46),(75,40),(59,34),(86,36),(97,45),(91,60),(81,77),(45,83),(16,93)]
]
def contour(points):
    out = [points[0]]
    for i in range(0,len(points)-1,3):
        for k in range(1,25):
            t=k/24
            weights=[(1-t)**3,3*t*(1-t)**2,3*t*t*(1-t),t**3]
            out.append(tuple(sum(weights[j]*points[i+j][a] for j in range(4)) for a in (0,1)))
    return out
print("static const unsigned char ribbon_mask[100][100] = {")
polygons=[contour(c) for c in curves]
for row in range(100):
    coverage=[0]*100
    for sy in (.25,.75):
        for polygon in polygons:
            edges=zip(polygon,polygon[1:])
            hits=sorted(a[0]+(row+sy-a[1])*(b[0]-a[0])/(b[1]-a[1]) for a,b in edges if (a[1]>row+sy)!=(b[1]>row+sy))
            for left,right in zip(hits[::2],hits[1::2]):
                for x in range(max(0,int(left)),min(100,int(right)+1)):
                    coverage[x]+=int(left<=x+.25<right)+int(left<=x+.75<right)
    print("{"+",".join(str(min(255,v*64)) for v in coverage)+"},")
print("};")
