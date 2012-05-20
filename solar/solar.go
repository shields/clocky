// Copyright 2012 Michael Shields
// 
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 
//     http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
 solar returns data about sunrise and sunset times.

 This implementation is a cheap hack and should be replace with a proper
 astronomical calcuation.

 Source data:
 http://www.usno.navy.mil/USNO/astronomical-applications/data-services/rs-one-year-us
*/
package solar

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// Rise returns the time of the next sunrise.  This may be on a following day,
// or many months later in the polar regions.
func Rise(t time.Time, lat, lng float64) time.Time {
	if math.Abs(lat-sfLat) > 0.5 || math.Abs(lng-sfLng) > 0.5 {
		panic("solar: only San Francisco is supported")
	}
	return next(t, &riseData)
}

func Set(t time.Time, lat, lng float64) time.Time {
	if math.Abs(lat-sfLat) > 0.5 || math.Abs(lng-sfLng) > 0.5 {
		panic("solar: only San Francisco is supported")
	}
	return next(t, &setData)
}

func next(t time.Time, data *[12][31]int) time.Time {
	midnight := time.Unix(t.Unix()-(t.Unix()-8*3600)%86400, 0).UTC()
	tt := time.Unix(midnight.Unix()+int64(data[t.Month()-1][t.Day()-1]), 0).UTC()
	if t.Unix() > tt.Unix() {
		return next(time.Unix(midnight.Unix()+86400, 0).UTC(), data)
	}
	return tt
}

// Seconds after PST (not PST/PDT) midnight.
var riseData, setData [12][31]int

func hhmm(hhmm string) int {
	h, _ := strconv.Atoi(hhmm[0:2])
	m, _ := strconv.Atoi(hhmm[2:4])
	return h*3600 + m*60
}

func init() {
	for _, line := range strings.Split(table, "\n") {
		if len(line) != 134 || line[0] < '0' || line[0] > '3' {
			continue
		}
		day, _ := strconv.Atoi(line[:2])
		for month := 0; month < 12; month++ {
			base := 4 + month*11
			riseData[month][day-1] = hhmm(line[base : base+4])
			setData[month][day-1] = hhmm(line[base+5 : base+9])
		}
	}
}

const sfLat, sfLng = (37 + 46./60), -(122 + 26./60)

const table = `
             o  ,    o  ,                              SAN FRANCISCO, CALIFORNIA                       Astronomical Applications Dept.
Location: W122 26, N37 46                          Rise and Set for the Sun for 2012                   U. S. Naval Observatory        
                                                                                                       Washington, DC  20392-5420     
                                                         Pacific Standard Time                                                        
                                                                                                                                      
                                                                                                                                      
       Jan.       Feb.       Mar.       Apr.       May        June       July       Aug.       Sept.      Oct.       Nov.       Dec.  
Day Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set  Rise  Set
     h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m   h m  h m
01  0725 1701  0714 1733  0640 1804  0554 1833  0513 1901  0449 1926  0452 1935  0514 1918  0540 1838  0606 1752  0636 1710  0707 1651
02  0725 1702  0713 1734  0639 1805  0553 1834  0512 1902  0449 1927  0452 1935  0515 1917  0541 1837  0607 1750  0637 1709  0708 1651
03  0725 1703  0712 1735  0638 1806  0551 1835  0511 1903  0449 1928  0453 1935  0516 1916  0542 1835  0608 1749  0638 1708  0709 1651
04  0725 1704  0711 1736  0636 1807  0550 1836  0510 1903  0448 1928  0453 1935  0516 1915  0543 1834  0609 1747  0639 1707  0710 1651
05  0726 1705  0710 1737  0635 1808  0548 1837  0509 1904  0448 1929  0454 1935  0517 1914  0544 1832  0609 1746  0640 1706  0711 1651
06  0726 1706  0709 1739  0633 1809  0547 1838  0508 1905  0448 1929  0455 1935  0518 1912  0545 1831  0610 1744  0641 1705  0711 1651
07  0725 1707  0709 1740  0632 1810  0545 1839  0507 1906  0448 1930  0455 1934  0519 1911  0545 1829  0611 1743  0642 1704  0712 1651
08  0725 1708  0708 1741  0630 1811  0544 1840  0506 1907  0448 1930  0456 1934  0520 1910  0546 1827  0612 1741  0643 1704  0713 1651
09  0725 1708  0706 1742  0629 1812  0542 1841  0505 1908  0448 1931  0456 1934  0521 1909  0547 1826  0613 1740  0644 1703  0714 1651
10  0725 1709  0705 1743  0627 1813  0541 1841  0504 1909  0447 1931  0457 1933  0522 1908  0548 1824  0614 1739  0645 1702  0715 1651
11  0725 1710  0704 1744  0626 1814  0540 1842  0503 1910  0447 1932  0458 1933  0522 1907  0549 1823  0615 1737  0646 1701  0715 1651
12  0725 1711  0703 1745  0624 1815  0538 1843  0502 1911  0447 1932  0458 1932  0523 1905  0550 1821  0616 1736  0647 1700  0716 1651
13  0725 1712  0702 1746  0623 1816  0537 1844  0501 1911  0447 1933  0459 1932  0524 1904  0550 1820  0617 1734  0648 1659  0717 1652
14  0724 1713  0701 1747  0621 1817  0535 1845  0500 1912  0447 1933  0500 1931  0525 1903  0551 1818  0618 1733  0650 1659  0718 1652
15  0724 1714  0700 1748  0620 1818  0534 1846  0459 1913  0447 1933  0500 1931  0526 1902  0552 1817  0619 1732  0651 1658  0718 1652
16  0724 1715  0659 1750  0618 1818  0532 1847  0459 1914  0448 1934  0501 1930  0527 1900  0553 1815  0620 1730  0652 1657  0719 1653
17  0723 1717  0657 1751  0617 1819  0531 1848  0458 1915  0448 1934  0502 1930  0528 1859  0554 1814  0621 1729  0653 1657  0720 1653
18  0723 1718  0656 1752  0615 1820  0530 1849  0457 1916  0448 1934  0503 1929  0528 1858  0555 1812  0622 1727  0654 1656  0720 1653
19  0723 1719  0655 1753  0614 1821  0528 1850  0456 1917  0448 1935  0503 1929  0529 1857  0556 1810  0623 1726  0655 1656  0721 1654
20  0722 1720  0654 1754  0612 1822  0527 1851  0456 1917  0448 1935  0504 1928  0530 1855  0556 1809  0624 1725  0656 1655  0721 1654
21  0722 1721  0653 1755  0611 1823  0526 1852  0455 1918  0448 1935  0505 1927  0531 1854  0557 1807  0625 1723  0657 1654  0722 1655
22  0721 1722  0651 1756  0609 1824  0524 1852  0454 1919  0449 1935  0506 1926  0532 1852  0558 1806  0626 1722  0658 1654  0722 1655
23  0720 1723  0650 1757  0608 1825  0523 1853  0454 1920  0449 1935  0506 1926  0533 1851  0559 1804  0627 1721  0659 1654  0723 1656
24  0720 1724  0649 1758  0606 1826  0522 1854  0453 1921  0449 1936  0507 1925  0534 1850  0600 1803  0628 1720  0700 1653  0723 1656
25  0719 1725  0647 1759  0605 1827  0521 1855  0453 1921  0449 1936  0508 1924  0534 1848  0601 1801  0629 1718  0701 1653  0723 1657
26  0719 1726  0646 1800  0603 1828  0519 1856  0452 1922  0450 1936  0509 1923  0535 1847  0601 1800  0630 1717  0702 1652  0724 1658
27  0718 1727  0645 1801  0602 1829  0518 1857  0452 1923  0450 1936  0510 1922  0536 1845  0602 1758  0631 1716  0703 1652  0724 1658
28  0717 1729  0643 1802  0600 1830  0517 1858  0451 1924  0451 1936  0510 1921  0537 1844  0603 1756  0632 1715  0704 1652  0724 1659
29  0716 1730  0642 1803  0559 1830  0516 1859  0451 1924  0451 1936  0511 1921  0538 1842  0604 1755  0633 1714  0705 1651  0725 1700
30  0716 1731             0557 1831  0515 1900  0450 1925  0451 1936  0512 1920  0539 1841  0605 1753  0634 1713  0706 1651  0725 1701
31  0715 1732             0556 1832             0450 1926             0513 1919  0540 1840             0635 1712             0725 1701

                                             Add one hour for daylight time, if and when in use.
`
