# Spoof Meta-Detector Experiment

Samples: 1000 (500 real, 500 synthetic)

## Holdout Result

- Evaluation: stratified holdout split, test_size=0.25, seed=7.
- Evaluated samples: 250/1000.
- best_zero_fp=thr=0.908, TPR=0.54, FPR=0.00, TP=67, FP=0, TN=125, FN=58; best_balanced=thr=0.658, TPR=0.91, FPR=0.03, TP=114, FP=4, TN=121, FN=11

## Scores

| file | label | split | holdout spoof score | in-sample score | current deepfake | current overall | latency ms |
|---|---:|---:|---:|---:|---:|---:|---:|
| `577.wav` | synthetic | eval | 0.993 | 0.993 |  |  |  |
| `655.wav` | synthetic | eval | 0.992 | 0.992 |  |  |  |
| `989.wav` | synthetic | eval | 0.986 | 0.986 |  |  |  |
| `451.wav` | synthetic | eval | 0.985 | 0.985 |  |  |  |
| `874.wav` | synthetic | eval | 0.985 | 0.983 |  |  |  |
| `999.wav` | synthetic | eval | 0.984 | 0.982 |  |  |  |
| `1227.wav` | synthetic | eval | 0.983 | 0.985 |  |  |  |
| `186.wav` | synthetic | eval | 0.982 | 0.979 |  |  |  |
| `25.wav` | synthetic | eval | 0.981 | 0.982 |  |  |  |
| `1082.wav` | synthetic | eval | 0.981 | 0.982 |  |  |  |
| `182.wav` | synthetic | eval | 0.981 | 0.980 |  |  |  |
| `17.wav` | synthetic | eval | 0.980 | 0.979 |  |  |  |
| `92.wav` | synthetic | eval | 0.979 | 0.971 |  |  |  |
| `251.wav` | synthetic | eval | 0.979 | 0.978 |  |  |  |
| `474.wav` | synthetic | eval | 0.977 | 0.979 |  |  |  |
| `81.wav` | synthetic | eval | 0.976 | 0.970 |  |  |  |
| `253.wav` | synthetic | eval | 0.976 | 0.977 |  |  |  |
| `1321.wav` | synthetic | eval | 0.975 | 0.968 |  |  |  |
| `374.wav` | synthetic | eval | 0.975 | 0.973 |  |  |  |
| `1110.wav` | synthetic | eval | 0.975 | 0.971 |  |  |  |
| `1115.wav` | synthetic | eval | 0.974 | 0.973 |  |  |  |
| `657.wav` | synthetic | eval | 0.974 | 0.974 |  |  |  |
| `1300.wav` | synthetic | eval | 0.974 | 0.974 |  |  |  |
| `1012.wav` | synthetic | eval | 0.974 | 0.970 |  |  |  |
| `593.wav` | synthetic | eval | 0.973 | 0.969 |  |  |  |
| `953.wav` | synthetic | eval | 0.972 | 0.965 |  |  |  |
| `215.wav` | synthetic | eval | 0.971 | 0.970 |  |  |  |
| `633.wav` | synthetic | eval | 0.971 | 0.971 |  |  |  |
| `1201.wav` | synthetic | eval | 0.970 | 0.964 |  |  |  |
| `1286.wav` | synthetic | eval | 0.970 | 0.967 |  |  |  |
| `527.wav` | synthetic | eval | 0.969 | 0.966 |  |  |  |
| `717.wav` | synthetic | eval | 0.967 | 0.969 |  |  |  |
| `560.wav` | synthetic | eval | 0.967 | 0.969 |  |  |  |
| `57.wav` | synthetic | eval | 0.964 | 0.957 |  |  |  |
| `996.wav` | synthetic | eval | 0.964 | 0.959 |  |  |  |
| `579.wav` | synthetic | eval | 0.964 | 0.966 |  |  |  |
| `770.wav` | synthetic | eval | 0.959 | 0.953 |  |  |  |
| `495.wav` | synthetic | eval | 0.957 | 0.946 |  |  |  |
| `1214.wav` | synthetic | eval | 0.955 | 0.949 |  |  |  |
| `372.wav` | synthetic | eval | 0.955 | 0.958 |  |  |  |
| `421.wav` | synthetic | eval | 0.953 | 0.955 |  |  |  |
| `489.wav` | synthetic | eval | 0.952 | 0.945 |  |  |  |
| `703.wav` | synthetic | eval | 0.951 | 0.945 |  |  |  |
| `278.wav` | synthetic | eval | 0.951 | 0.953 |  |  |  |
| `21.wav` | synthetic | eval | 0.949 | 0.947 |  |  |  |
| `1297.wav` | synthetic | eval | 0.947 | 0.957 |  |  |  |
| `546.wav` | synthetic | eval | 0.946 | 0.938 |  |  |  |
| `1150.wav` | synthetic | eval | 0.946 | 0.957 |  |  |  |
| `257.wav` | synthetic | eval | 0.945 | 0.950 |  |  |  |
| `103.wav` | synthetic | eval | 0.944 | 0.939 |  |  |  |
| `69.wav` | synthetic | eval | 0.942 | 0.933 |  |  |  |
| `605.wav` | synthetic | eval | 0.939 | 0.927 |  |  |  |
| `609.wav` | synthetic | eval | 0.936 | 0.938 |  |  |  |
| `835.wav` | synthetic | eval | 0.935 | 0.928 |  |  |  |
| `3.wav` | synthetic | eval | 0.932 | 0.944 |  |  |  |
| `549.wav` | synthetic | eval | 0.926 | 0.916 |  |  |  |
| `1106.wav` | synthetic | eval | 0.925 | 0.926 |  |  |  |
| `1191.wav` | synthetic | eval | 0.925 | 0.929 |  |  |  |
| `128.wav` | synthetic | eval | 0.921 | 0.929 |  |  |  |
| `764.wav` | synthetic | eval | 0.916 | 0.935 |  |  |  |
| `977.wav` | synthetic | eval | 0.915 | 0.907 |  |  |  |
| `621.wav` | synthetic | eval | 0.913 | 0.903 |  |  |  |
| `1136.wav` | synthetic | eval | 0.911 | 0.909 |  |  |  |
| `1290.wav` | synthetic | eval | 0.911 | 0.896 |  |  |  |
| `1315.wav` | synthetic | eval | 0.909 | 0.905 |  |  |  |
| `292.wav` | synthetic | eval | 0.908 | 0.893 |  |  |  |
| `893.wav` | synthetic | eval | 0.908 | 0.919 |  |  |  |
| `691.wav` | real | eval | 0.907 | 0.881 |  |  |  |
| `1263.wav` | synthetic | eval | 0.905 | 0.924 |  |  |  |
| `466.wav` | synthetic | eval | 0.901 | 0.920 |  |  |  |
| `651.wav` | synthetic | eval | 0.891 | 0.879 |  |  |  |
| `1332.wav` | synthetic | eval | 0.891 | 0.895 |  |  |  |
| `35.wav` | synthetic | eval | 0.889 | 0.865 |  |  |  |
| `1051.wav` | synthetic | eval | 0.889 | 0.897 |  |  |  |
| `395.wav` | synthetic | eval | 0.883 | 0.906 |  |  |  |
| `769.wav` | synthetic | eval | 0.879 | 0.894 |  |  |  |
| `1258.wav` | synthetic | eval | 0.877 | 0.874 |  |  |  |
| `677.wav` | synthetic | eval | 0.877 | 0.887 |  |  |  |
| `68.wav` | synthetic | eval | 0.874 | 0.864 |  |  |  |
| `172.wav` | synthetic | eval | 0.864 | 0.860 |  |  |  |
| `970.wav` | synthetic | eval | 0.862 | 0.847 |  |  |  |
| `822.wav` | synthetic | eval | 0.859 | 0.856 |  |  |  |
| `972.wav` | synthetic | eval | 0.858 | 0.858 |  |  |  |
| `502.wav` | synthetic | eval | 0.856 | 0.865 |  |  |  |
| `840.wav` | synthetic | eval | 0.855 | 0.839 |  |  |  |
| `390.wav` | synthetic | eval | 0.855 | 0.826 |  |  |  |
| `1350.wav` | synthetic | eval | 0.855 | 0.873 |  |  |  |
| `441.wav` | synthetic | eval | 0.854 | 0.813 |  |  |  |
| `341.wav` | synthetic | eval | 0.851 | 0.862 |  |  |  |
| `1014.wav` | synthetic | eval | 0.851 | 0.853 |  |  |  |
| `101.wav` | synthetic | eval | 0.847 | 0.814 |  |  |  |
| `1048.wav` | synthetic | eval | 0.836 | 0.855 |  |  |  |
| `538.wav` | synthetic | eval | 0.832 | 0.849 |  |  |  |
| `1030.wav` | synthetic | eval | 0.831 | 0.802 |  |  |  |
| `481.wav` | synthetic | eval | 0.830 | 0.865 |  |  |  |
| `573.wav` | synthetic | eval | 0.829 | 0.838 |  |  |  |
| `1184.wav` | synthetic | eval | 0.826 | 0.797 |  |  |  |
| `135.wav` | synthetic | eval | 0.810 | 0.767 |  |  |  |
| `519.wav` | synthetic | eval | 0.807 | 0.841 |  |  |  |
| `616.wav` | synthetic | eval | 0.805 | 0.803 |  |  |  |
| `303.wav` | synthetic | eval | 0.804 | 0.766 |  |  |  |
| `715.wav` | synthetic | eval | 0.794 | 0.752 |  |  |  |
| `943.wav` | synthetic | eval | 0.768 | 0.756 |  |  |  |
| `1008.wav` | synthetic | eval | 0.766 | 0.810 |  |  |  |
| `647.wav` | synthetic | eval | 0.756 | 0.768 |  |  |  |
| `772.wav` | real | eval | 0.741 | 0.730 |  |  |  |
| `297.wav` | synthetic | eval | 0.740 | 0.752 |  |  |  |
| `1225.wav` | synthetic | eval | 0.738 | 0.708 |  |  |  |
| `267.wav` | synthetic | eval | 0.733 | 0.757 |  |  |  |
| `473.wav` | real | eval | 0.711 | 0.689 |  |  |  |
| `163.wav` | synthetic | eval | 0.707 | 0.725 |  |  |  |
| `505.wav` | synthetic | eval | 0.700 | 0.676 |  |  |  |
| `1122.wav` | synthetic | eval | 0.693 | 0.696 |  |  |  |
| `852.wav` | synthetic | eval | 0.680 | 0.696 |  |  |  |
| `645.wav` | real | eval | 0.680 | 0.674 |  |  |  |
| `1282.wav` | synthetic | eval | 0.679 | 0.655 |  |  |  |
| `864.wav` | synthetic | eval | 0.665 | 0.666 |  |  |  |
| `881.wav` | synthetic | eval | 0.658 | 0.630 |  |  |  |
| `87.wav` | real | eval | 0.633 | 0.668 |  |  |  |
| `45.wav` | real | eval | 0.629 | 0.650 |  |  |  |
| `301.wav` | real | eval | 0.611 | 0.589 |  |  |  |
| `162.wav` | real | eval | 0.574 | 0.579 |  |  |  |
| `589.wav` | real | eval | 0.565 | 0.559 |  |  |  |
| `334.wav` | real | eval | 0.549 | 0.503 |  |  |  |
| `277.wav` | synthetic | eval | 0.514 | 0.512 |  |  |  |
| `368.wav` | synthetic | eval | 0.444 | 0.438 |  |  |  |
| `622.wav` | real | eval | 0.431 | 0.402 |  |  |  |
| `553.wav` | real | eval | 0.430 | 0.430 |  |  |  |
| `624.wav` | real | eval | 0.421 | 0.450 |  |  |  |
| `72.wav` | real | eval | 0.389 | 0.355 |  |  |  |
| `305.wav` | real | eval | 0.383 | 0.443 |  |  |  |
| `306.wav` | real | eval | 0.373 | 0.402 |  |  |  |
| `744.wav` | real | eval | 0.364 | 0.420 |  |  |  |
| `656.wav` | real | eval | 0.361 | 0.357 |  |  |  |
| `325.wav` | real | eval | 0.354 | 0.327 |  |  |  |
| `364.wav` | real | eval | 0.350 | 0.413 |  |  |  |
| `520.wav` | real | eval | 0.332 | 0.299 |  |  |  |
| `177.wav` | real | eval | 0.315 | 0.264 |  |  |  |
| `85.wav` | real | eval | 0.313 | 0.339 |  |  |  |
| `123.wav` | real | eval | 0.304 | 0.320 |  |  |  |
| `358.wav` | real | eval | 0.301 | 0.274 |  |  |  |
| `331.wav` | real | eval | 0.299 | 0.275 |  |  |  |
| `853.wav` | synthetic | eval | 0.293 | 0.363 |  |  |  |
| `615.wav` | real | eval | 0.289 | 0.280 |  |  |  |
| `668.wav` | real | eval | 0.287 | 0.278 |  |  |  |
| `221.wav` | real | eval | 0.279 | 0.279 |  |  |  |
| `469.wav` | real | eval | 0.278 | 0.267 |  |  |  |
| `482.wav` | real | eval | 0.255 | 0.275 |  |  |  |
| `486.wav` | real | eval | 0.255 | 0.234 |  |  |  |
| `396.wav` | real | eval | 0.247 | 0.274 |  |  |  |
| `412.wav` | real | eval | 0.241 | 0.209 |  |  |  |
| `786.wav` | real | eval | 0.236 | 0.259 |  |  |  |
| `352.wav` | real | eval | 0.234 | 0.235 |  |  |  |
| `371.wav` | real | eval | 0.232 | 0.185 |  |  |  |
| `650.wav` | real | eval | 0.226 | 0.245 |  |  |  |
| `745.wav` | real | eval | 0.216 | 0.221 |  |  |  |
| `180.wav` | real | eval | 0.213 | 0.226 |  |  |  |
| `405.wav` | real | eval | 0.212 | 0.209 |  |  |  |
| `327.wav` | real | eval | 0.210 | 0.180 |  |  |  |
| `788.wav` | real | eval | 0.196 | 0.213 |  |  |  |
| `238.wav` | real | eval | 0.191 | 0.200 |  |  |  |
| `729.wav` | real | eval | 0.190 | 0.266 |  |  |  |
| `311.wav` | real | eval | 0.180 | 0.153 |  |  |  |
| `791.wav` | synthetic | eval | 0.167 | 0.210 |  |  |  |
| `7.wav` | real | eval | 0.166 | 0.191 |  |  |  |
| `112.wav` | real | eval | 0.164 | 0.149 |  |  |  |
| `683.wav` | real | eval | 0.161 | 0.157 |  |  |  |
| `658.wav` | real | eval | 0.158 | 0.167 |  |  |  |
| `157.wav` | real | eval | 0.156 | 0.173 |  |  |  |
| `102.wav` | real | eval | 0.144 | 0.164 |  |  |  |
| `142.wav` | real | eval | 0.127 | 0.124 |  |  |  |
| `330.wav` | real | eval | 0.119 | 0.111 |  |  |  |
| `298.wav` | real | eval | 0.117 | 0.156 |  |  |  |
| `724.wav` | real | eval | 0.113 | 0.111 |  |  |  |
| `95.wav` | real | eval | 0.108 | 0.101 |  |  |  |
| `731.wav` | real | eval | 0.094 | 0.105 |  |  |  |
| `90.wav` | real | eval | 0.094 | 0.086 |  |  |  |
| `785.wav` | real | eval | 0.092 | 0.107 |  |  |  |
| `654.wav` | real | eval | 0.090 | 0.121 |  |  |  |
| `713.wav` | real | eval | 0.085 | 0.081 |  |  |  |
| `229.wav` | synthetic | eval | 0.078 | 0.096 |  |  |  |
| `575.wav` | real | eval | 0.073 | 0.078 |  |  |  |
| `243.wav` | real | eval | 0.072 | 0.088 |  |  |  |
| `378.wav` | real | eval | 0.070 | 0.064 |  |  |  |
| `216.wav` | real | eval | 0.070 | 0.065 |  |  |  |
| `1237.wav` | synthetic | eval | 0.067 | 0.090 |  |  |  |
| `209.wav` | synthetic | eval | 0.064 | 0.078 |  |  |  |
| `780.wav` | real | eval | 0.063 | 0.066 |  |  |  |
| `1020.wav` | synthetic | eval | 0.061 | 0.073 |  |  |  |
| `37.wav` | real | eval | 0.060 | 0.077 |  |  |  |
| `613.wav` | real | eval | 0.059 | 0.075 |  |  |  |
| `206.wav` | real | eval | 0.058 | 0.050 |  |  |  |
| `241.wav` | real | eval | 0.057 | 0.078 |  |  |  |
| `234.wav` | real | eval | 0.057 | 0.061 |  |  |  |
| `508.wav` | real | eval | 0.056 | 0.059 |  |  |  |
| `167.wav` | real | eval | 0.051 | 0.052 |  |  |  |
| `507.wav` | real | eval | 0.048 | 0.049 |  |  |  |
| `1114.wav` | synthetic | eval | 0.048 | 0.051 |  |  |  |
| `255.wav` | real | eval | 0.048 | 0.069 |  |  |  |
| `757.wav` | real | eval | 0.046 | 0.049 |  |  |  |
| `735.wav` | real | eval | 0.045 | 0.054 |  |  |  |
| `759.wav` | real | eval | 0.036 | 0.028 |  |  |  |
| `449.wav` | real | eval | 0.036 | 0.039 |  |  |  |
| `152.wav` | real | eval | 0.035 | 0.038 |  |  |  |
| `360.wav` | real | eval | 0.034 | 0.029 |  |  |  |
| `192.wav` | real | eval | 0.032 | 0.036 |  |  |  |
| `401.wav` | real | eval | 0.029 | 0.029 |  |  |  |
| `410.wav` | real | eval | 0.028 | 0.029 |  |  |  |
| `96.wav` | real | eval | 0.027 | 0.036 |  |  |  |
| `686.wav` | real | eval | 0.025 | 0.026 |  |  |  |
| `576.wav` | real | eval | 0.024 | 0.031 |  |  |  |
| `411.wav` | real | eval | 0.023 | 0.020 |  |  |  |
| `178.wav` | real | eval | 0.022 | 0.023 |  |  |  |
| `93.wav` | real | eval | 0.021 | 0.029 |  |  |  |
| `343.wav` | real | eval | 0.020 | 0.026 |  |  |  |
| `20.wav` | real | eval | 0.015 | 0.020 |  |  |  |
| `130.wav` | real | eval | 0.015 | 0.013 |  |  |  |
| `404.wav` | real | eval | 0.015 | 0.013 |  |  |  |
| `739.wav` | real | eval | 0.013 | 0.015 |  |  |  |
| `111.wav` | real | eval | 0.013 | 0.010 |  |  |  |
| `678.wav` | real | eval | 0.012 | 0.012 |  |  |  |
| `636.wav` | synthetic | eval | 0.010 | 0.020 |  |  |  |
| `409.wav` | real | eval | 0.010 | 0.009 |  |  |  |
| `400.wav` | real | eval | 0.008 | 0.008 |  |  |  |
| `160.wav` | real | eval | 0.008 | 0.009 |  |  |  |
| `483.wav` | real | eval | 0.008 | 0.007 |  |  |  |
| `632.wav` | real | eval | 0.007 | 0.008 |  |  |  |
| `461.wav` | real | eval | 0.007 | 0.008 |  |  |  |
| `733.wav` | real | eval | 0.006 | 0.009 |  |  |  |
| `185.wav` | real | eval | 0.005 | 0.007 |  |  |  |
| `626.wav` | real | eval | 0.005 | 0.004 |  |  |  |
| `77.wav` | real | eval | 0.004 | 0.005 |  |  |  |
| `497.wav` | real | eval | 0.004 | 0.005 |  |  |  |
| `467.wav` | real | eval | 0.004 | 0.004 |  |  |  |
| `595.wav` | real | eval | 0.003 | 0.004 |  |  |  |
| `385.wav` | real | eval | 0.003 | 0.005 |  |  |  |
| `4.wav` | real | eval | 0.003 | 0.004 |  |  |  |
| `147.wav` | real | eval | 0.003 | 0.004 |  |  |  |
| `248.wav` | real | eval | 0.002 | 0.003 |  |  |  |
| `321.wav` | real | eval | 0.002 | 0.002 |  |  |  |
| `598.wav` | real | eval | 0.002 | 0.002 |  |  |  |
| `133.wav` | real | eval | 0.001 | 0.001 |  |  |  |
| `429.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `522.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `173.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `716.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `291.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `376.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `28.wav` | real | eval | 0.000 | 0.000 |  |  |  |
| `122.wav` | synthetic | eval | 0.000 | 0.000 |  |  |  |
| `0.wav` | synthetic | train |  | 0.838 |  |  |  |
| `1.wav` | synthetic | train |  | 0.981 |  |  |  |
| `2.wav` | synthetic | train |  | 0.867 |  |  |  |
| `5.wav` | real | train |  | 0.034 |  |  |  |
| `6.wav` | synthetic | train |  | 0.979 |  |  |  |
| `8.wav` | real | train |  | 0.309 |  |  |  |
| `9.wav` | real | train |  | 0.353 |  |  |  |
| `10.wav` | synthetic | train |  | 0.959 |  |  |  |
| `11.wav` | real | train |  | 0.463 |  |  |  |
| `12.wav` | real | train |  | 0.187 |  |  |  |
| `13.wav` | synthetic | train |  | 0.827 |  |  |  |
| `14.wav` | real | train |  | 0.000 |  |  |  |
| `15.wav` | synthetic | train |  | 0.932 |  |  |  |
| `16.wav` | synthetic | train |  | 0.903 |  |  |  |
| `18.wav` | real | train |  | 0.003 |  |  |  |
| `19.wav` | synthetic | train |  | 0.978 |  |  |  |
| `22.wav` | real | train |  | 0.187 |  |  |  |
| `23.wav` | real | train |  | 0.080 |  |  |  |
| `24.wav` | synthetic | train |  | 0.742 |  |  |  |
| `26.wav` | real | train |  | 0.200 |  |  |  |
| `27.wav` | real | train |  | 0.090 |  |  |  |
| `29.wav` | real | train |  | 0.143 |  |  |  |
| `30.wav` | real | train |  | 0.000 |  |  |  |
| `31.wav` | synthetic | train |  | 0.819 |  |  |  |
| `32.wav` | synthetic | train |  | 0.721 |  |  |  |
| `33.wav` | real | train |  | 0.437 |  |  |  |
| `34.wav` | synthetic | train |  | 0.532 |  |  |  |
| `36.wav` | real | train |  | 0.033 |  |  |  |
| `38.wav` | real | train |  | 0.380 |  |  |  |
| `39.wav` | real | train |  | 0.000 |  |  |  |
| `40.wav` | synthetic | train |  | 0.412 |  |  |  |
| `41.wav` | synthetic | train |  | 0.940 |  |  |  |
| `42.wav` | synthetic | train |  | 0.873 |  |  |  |
| `43.wav` | synthetic | train |  | 0.587 |  |  |  |
| `44.wav` | real | train |  | 0.245 |  |  |  |
| `46.wav` | real | train |  | 0.470 |  |  |  |
| `47.wav` | synthetic | train |  | 0.849 |  |  |  |
| `48.wav` | real | train |  | 0.130 |  |  |  |
| `49.wav` | real | train |  | 0.361 |  |  |  |
| `50.wav` | synthetic | train |  | 0.838 |  |  |  |
| `51.wav` | real | train |  | 0.032 |  |  |  |
| `52.wav` | real | train |  | 0.163 |  |  |  |
| `53.wav` | real | train |  | 0.046 |  |  |  |
| `54.wav` | real | train |  | 0.000 |  |  |  |
| `55.wav` | real | train |  | 0.006 |  |  |  |
| `56.wav` | real | train |  | 0.318 |  |  |  |
| `58.wav` | real | train |  | 0.006 |  |  |  |
| `59.wav` | synthetic | train |  | 0.914 |  |  |  |
| `60.wav` | synthetic | train |  | 0.812 |  |  |  |
| `61.wav` | synthetic | train |  | 0.960 |  |  |  |
| `62.wav` | synthetic | train |  | 0.925 |  |  |  |
| `63.wav` | synthetic | train |  | 0.758 |  |  |  |
| `64.wav` | real | train |  | 0.103 |  |  |  |
| `65.wav` | synthetic | train |  | 0.969 |  |  |  |
| `66.wav` | real | train |  | 0.000 |  |  |  |
| `67.wav` | real | train |  | 0.191 |  |  |  |
| `70.wav` | real | train |  | 0.557 |  |  |  |
| `71.wav` | real | train |  | 0.054 |  |  |  |
| `73.wav` | real | train |  | 0.067 |  |  |  |
| `74.wav` | real | train |  | 0.020 |  |  |  |
| `75.wav` | real | train |  | 0.002 |  |  |  |
| `76.wav` | synthetic | train |  | 0.913 |  |  |  |
| `78.wav` | real | train |  | 0.092 |  |  |  |
| `79.wav` | synthetic | train |  | 0.225 |  |  |  |
| `80.wav` | real | train |  | 0.369 |  |  |  |
| `82.wav` | real | train |  | 0.000 |  |  |  |
| `83.wav` | real | train |  | 0.207 |  |  |  |
| `84.wav` | real | train |  | 0.001 |  |  |  |
| `86.wav` | real | train |  | 0.230 |  |  |  |
| `88.wav` | real | train |  | 0.560 |  |  |  |
| `89.wav` | real | train |  | 0.116 |  |  |  |
| `91.wav` | synthetic | train |  | 0.800 |  |  |  |
| `94.wav` | real | train |  | 0.184 |  |  |  |
| `97.wav` | real | train |  | 0.423 |  |  |  |
| `98.wav` | real | train |  | 0.000 |  |  |  |
| `99.wav` | real | train |  | 0.042 |  |  |  |
| `100.wav` | real | train |  | 0.207 |  |  |  |
| `104.wav` | real | train |  | 0.076 |  |  |  |
| `105.wav` | real | train |  | 0.014 |  |  |  |
| `106.wav` | real | train |  | 0.899 |  |  |  |
| `107.wav` | synthetic | train |  | 0.951 |  |  |  |
| `108.wav` | real | train |  | 0.019 |  |  |  |
| `109.wav` | real | train |  | 0.012 |  |  |  |
| `110.wav` | synthetic | train |  | 0.958 |  |  |  |
| `113.wav` | real | train |  | 0.199 |  |  |  |
| `114.wav` | synthetic | train |  | 0.851 |  |  |  |
| `115.wav` | real | train |  | 0.068 |  |  |  |
| `116.wav` | synthetic | train |  | 0.988 |  |  |  |
| `117.wav` | synthetic | train |  | 0.991 |  |  |  |
| `118.wav` | synthetic | train |  | 0.970 |  |  |  |
| `119.wav` | real | train |  | 0.178 |  |  |  |
| `120.wav` | real | train |  | 0.378 |  |  |  |
| `121.wav` | real | train |  | 0.013 |  |  |  |
| `124.wav` | real | train |  | 0.286 |  |  |  |
| `125.wav` | real | train |  | 0.062 |  |  |  |
| `126.wav` | synthetic | train |  | 0.943 |  |  |  |
| `127.wav` | synthetic | train |  | 0.935 |  |  |  |
| `129.wav` | real | train |  | 0.117 |  |  |  |
| `131.wav` | real | train |  | 0.428 |  |  |  |
| `132.wav` | synthetic | train |  | 0.896 |  |  |  |
| `134.wav` | real | train |  | 0.298 |  |  |  |
| `136.wav` | synthetic | train |  | 0.977 |  |  |  |
| `137.wav` | real | train |  | 0.279 |  |  |  |
| `138.wav` | synthetic | train |  | 0.980 |  |  |  |
| `139.wav` | synthetic | train |  | 0.940 |  |  |  |
| `140.wav` | real | train |  | 0.319 |  |  |  |
| `141.wav` | real | train |  | 0.022 |  |  |  |
| `143.wav` | real | train |  | 0.105 |  |  |  |
| `144.wav` | synthetic | train |  | 0.851 |  |  |  |
| `145.wav` | real | train |  | 0.174 |  |  |  |
| `146.wav` | synthetic | train |  | 0.360 |  |  |  |
| `148.wav` | synthetic | train |  | 0.086 |  |  |  |
| `149.wav` | real | train |  | 0.003 |  |  |  |
| `150.wav` | synthetic | train |  | 0.873 |  |  |  |
| `151.wav` | real | train |  | 0.001 |  |  |  |
| `153.wav` | synthetic | train |  | 0.432 |  |  |  |
| `154.wav` | real | train |  | 0.073 |  |  |  |
| `155.wav` | synthetic | train |  | 0.786 |  |  |  |
| `156.wav` | real | train |  | 0.140 |  |  |  |
| `158.wav` | real | train |  | 0.383 |  |  |  |
| `159.wav` | synthetic | train |  | 0.858 |  |  |  |
| `161.wav` | real | train |  | 0.505 |  |  |  |
| `164.wav` | real | train |  | 0.010 |  |  |  |
| `165.wav` | synthetic | train |  | 0.942 |  |  |  |
| `166.wav` | real | train |  | 0.032 |  |  |  |
| `168.wav` | real | train |  | 0.712 |  |  |  |
| `169.wav` | real | train |  | 0.028 |  |  |  |
| `170.wav` | synthetic | train |  | 0.930 |  |  |  |
| `171.wav` | real | train |  | 0.509 |  |  |  |
| `174.wav` | synthetic | train |  | 0.961 |  |  |  |
| `175.wav` | synthetic | train |  | 0.023 |  |  |  |
| `176.wav` | real | train |  | 0.009 |  |  |  |
| `179.wav` | synthetic | train |  | 0.926 |  |  |  |
| `181.wav` | real | train |  | 0.024 |  |  |  |
| `183.wav` | real | train |  | 0.050 |  |  |  |
| `184.wav` | synthetic | train |  | 0.852 |  |  |  |
| `187.wav` | real | train |  | 0.093 |  |  |  |
| `188.wav` | synthetic | train |  | 0.983 |  |  |  |
| `189.wav` | real | train |  | 0.002 |  |  |  |
| `190.wav` | synthetic | train |  | 0.913 |  |  |  |
| `191.wav` | synthetic | train |  | 0.990 |  |  |  |
| `193.wav` | synthetic | train |  | 0.894 |  |  |  |
| `194.wav` | real | train |  | 0.092 |  |  |  |
| `195.wav` | real | train |  | 0.002 |  |  |  |
| `196.wav` | real | train |  | 0.001 |  |  |  |
| `197.wav` | real | train |  | 0.275 |  |  |  |
| `198.wav` | real | train |  | 0.004 |  |  |  |
| `199.wav` | real | train |  | 0.107 |  |  |  |
| `200.wav` | real | train |  | 0.329 |  |  |  |
| `201.wav` | real | train |  | 0.669 |  |  |  |
| `202.wav` | real | train |  | 0.012 |  |  |  |
| `203.wav` | real | train |  | 0.248 |  |  |  |
| `204.wav` | real | train |  | 0.054 |  |  |  |
| `205.wav` | real | train |  | 0.026 |  |  |  |
| `207.wav` | synthetic | train |  | 0.801 |  |  |  |
| `208.wav` | real | train |  | 0.001 |  |  |  |
| `210.wav` | real | train |  | 0.004 |  |  |  |
| `211.wav` | synthetic | train |  | 0.815 |  |  |  |
| `212.wav` | real | train |  | 0.001 |  |  |  |
| `213.wav` | real | train |  | 0.758 |  |  |  |
| `214.wav` | synthetic | train |  | 0.917 |  |  |  |
| `217.wav` | real | train |  | 0.067 |  |  |  |
| `218.wav` | real | train |  | 0.026 |  |  |  |
| `219.wav` | real | train |  | 0.257 |  |  |  |
| `220.wav` | real | train |  | 0.304 |  |  |  |
| `222.wav` | real | train |  | 0.274 |  |  |  |
| `223.wav` | real | train |  | 0.489 |  |  |  |
| `224.wav` | synthetic | train |  | 0.957 |  |  |  |
| `225.wav` | synthetic | train |  | 0.907 |  |  |  |
| `226.wav` | synthetic | train |  | 0.398 |  |  |  |
| `227.wav` | real | train |  | 0.320 |  |  |  |
| `228.wav` | real | train |  | 0.146 |  |  |  |
| `230.wav` | real | train |  | 0.064 |  |  |  |
| `231.wav` | synthetic | train |  | 0.720 |  |  |  |
| `232.wav` | real | train |  | 0.445 |  |  |  |
| `233.wav` | real | train |  | 0.032 |  |  |  |
| `235.wav` | real | train |  | 0.033 |  |  |  |
| `236.wav` | synthetic | train |  | 0.985 |  |  |  |
| `237.wav` | real | train |  | 0.122 |  |  |  |
| `239.wav` | real | train |  | 0.680 |  |  |  |
| `240.wav` | real | train |  | 0.018 |  |  |  |
| `242.wav` | real | train |  | 0.031 |  |  |  |
| `244.wav` | real | train |  | 0.180 |  |  |  |
| `245.wav` | real | train |  | 0.016 |  |  |  |
| `246.wav` | synthetic | train |  | 0.888 |  |  |  |
| `247.wav` | real | train |  | 0.179 |  |  |  |
| `249.wav` | real | train |  | 0.322 |  |  |  |
| `250.wav` | real | train |  | 0.423 |  |  |  |
| `252.wav` | real | train |  | 0.000 |  |  |  |
| `254.wav` | real | train |  | 0.409 |  |  |  |
| `256.wav` | real | train |  | 0.000 |  |  |  |
| `258.wav` | real | train |  | 0.075 |  |  |  |
| `259.wav` | real | train |  | 0.001 |  |  |  |
| `260.wav` | real | train |  | 0.132 |  |  |  |
| `261.wav` | synthetic | train |  | 0.945 |  |  |  |
| `262.wav` | real | train |  | 0.014 |  |  |  |
| `263.wav` | real | train |  | 0.145 |  |  |  |
| `264.wav` | real | train |  | 0.508 |  |  |  |
| `265.wav` | synthetic | train |  | 0.950 |  |  |  |
| `266.wav` | real | train |  | 0.142 |  |  |  |
| `268.wav` | real | train |  | 0.052 |  |  |  |
| `269.wav` | synthetic | train |  | 0.900 |  |  |  |
| `270.wav` | synthetic | train |  | 0.932 |  |  |  |
| `271.wav` | synthetic | train |  | 0.923 |  |  |  |
| `272.wav` | real | train |  | 0.333 |  |  |  |
| `273.wav` | synthetic | train |  | 0.883 |  |  |  |
| `274.wav` | real | train |  | 0.115 |  |  |  |
| `275.wav` | synthetic | train |  | 0.972 |  |  |  |
| `276.wav` | real | train |  | 0.370 |  |  |  |
| `279.wav` | real | train |  | 0.013 |  |  |  |
| `280.wav` | real | train |  | 0.003 |  |  |  |
| `281.wav` | real | train |  | 0.313 |  |  |  |
| `282.wav` | synthetic | train |  | 0.908 |  |  |  |
| `283.wav` | synthetic | train |  | 0.953 |  |  |  |
| `284.wav` | real | train |  | 0.042 |  |  |  |
| `285.wav` | synthetic | train |  | 0.993 |  |  |  |
| `286.wav` | real | train |  | 0.365 |  |  |  |
| `287.wav` | synthetic | train |  | 0.853 |  |  |  |
| `288.wav` | real | train |  | 0.399 |  |  |  |
| `289.wav` | real | train |  | 0.108 |  |  |  |
| `290.wav` | real | train |  | 0.181 |  |  |  |
| `293.wav` | synthetic | train |  | 0.944 |  |  |  |
| `294.wav` | synthetic | train |  | 0.933 |  |  |  |
| `295.wav` | synthetic | train |  | 0.712 |  |  |  |
| `296.wav` | synthetic | train |  | 0.761 |  |  |  |
| `299.wav` | real | train |  | 0.431 |  |  |  |
| `300.wav` | real | train |  | 0.363 |  |  |  |
| `302.wav` | synthetic | train |  | 0.889 |  |  |  |
| `304.wav` | real | train |  | 0.347 |  |  |  |
| `307.wav` | synthetic | train |  | 0.962 |  |  |  |
| `308.wav` | real | train |  | 0.008 |  |  |  |
| `309.wav` | real | train |  | 0.374 |  |  |  |
| `310.wav` | synthetic | train |  | 0.982 |  |  |  |
| `312.wav` | synthetic | train |  | 0.970 |  |  |  |
| `313.wav` | real | train |  | 0.356 |  |  |  |
| `314.wav` | real | train |  | 0.044 |  |  |  |
| `315.wav` | real | train |  | 0.176 |  |  |  |
| `316.wav` | real | train |  | 0.091 |  |  |  |
| `317.wav` | synthetic | train |  | 0.195 |  |  |  |
| `318.wav` | synthetic | train |  | 0.771 |  |  |  |
| `319.wav` | synthetic | train |  | 0.857 |  |  |  |
| `320.wav` | real | train |  | 0.269 |  |  |  |
| `322.wav` | real | train |  | 0.285 |  |  |  |
| `323.wav` | real | train |  | 0.118 |  |  |  |
| `324.wav` | synthetic | train |  | 0.951 |  |  |  |
| `326.wav` | real | train |  | 0.013 |  |  |  |
| `328.wav` | real | train |  | 0.474 |  |  |  |
| `329.wav` | real | train |  | 0.783 |  |  |  |
| `332.wav` | real | train |  | 0.765 |  |  |  |
| `333.wav` | real | train |  | 0.000 |  |  |  |
| `335.wav` | real | train |  | 0.414 |  |  |  |
| `336.wav` | real | train |  | 0.030 |  |  |  |
| `337.wav` | real | train |  | 0.000 |  |  |  |
| `338.wav` | synthetic | train |  | 0.847 |  |  |  |
| `339.wav` | synthetic | train |  | 0.594 |  |  |  |
| `340.wav` | real | train |  | 0.171 |  |  |  |
| `342.wav` | synthetic | train |  | 0.644 |  |  |  |
| `344.wav` | synthetic | train |  | 0.436 |  |  |  |
| `345.wav` | real | train |  | 0.429 |  |  |  |
| `346.wav` | synthetic | train |  | 0.918 |  |  |  |
| `347.wav` | synthetic | train |  | 0.904 |  |  |  |
| `348.wav` | real | train |  | 0.350 |  |  |  |
| `349.wav` | synthetic | train |  | 0.892 |  |  |  |
| `350.wav` | real | train |  | 0.000 |  |  |  |
| `351.wav` | synthetic | train |  | 0.910 |  |  |  |
| `353.wav` | synthetic | train |  | 0.934 |  |  |  |
| `354.wav` | synthetic | train |  | 0.976 |  |  |  |
| `355.wav` | real | train |  | 0.388 |  |  |  |
| `356.wav` | real | train |  | 0.538 |  |  |  |
| `357.wav` | synthetic | train |  | 0.865 |  |  |  |
| `359.wav` | real | train |  | 0.071 |  |  |  |
| `361.wav` | synthetic | train |  | 0.889 |  |  |  |
| `362.wav` | real | train |  | 0.456 |  |  |  |
| `363.wav` | real | train |  | 0.134 |  |  |  |
| `365.wav` | real | train |  | 0.178 |  |  |  |
| `366.wav` | real | train |  | 0.108 |  |  |  |
| `367.wav` | real | train |  | 0.004 |  |  |  |
| `369.wav` | real | train |  | 0.155 |  |  |  |
| `370.wav` | real | train |  | 0.012 |  |  |  |
| `373.wav` | real | train |  | 0.009 |  |  |  |
| `375.wav` | synthetic | train |  | 0.813 |  |  |  |
| `377.wav` | real | train |  | 0.032 |  |  |  |
| `379.wav` | real | train |  | 0.000 |  |  |  |
| `380.wav` | synthetic | train |  | 0.946 |  |  |  |
| `381.wav` | real | train |  | 0.002 |  |  |  |
| `382.wav` | real | train |  | 0.080 |  |  |  |
| `383.wav` | real | train |  | 0.507 |  |  |  |
| `384.wav` | real | train |  | 0.004 |  |  |  |
| `386.wav` | real | train |  | 0.001 |  |  |  |
| `387.wav` | real | train |  | 0.270 |  |  |  |
| `388.wav` | real | train |  | 0.651 |  |  |  |
| `389.wav` | real | train |  | 0.075 |  |  |  |
| `391.wav` | synthetic | train |  | 0.253 |  |  |  |
| `392.wav` | synthetic | train |  | 0.000 |  |  |  |
| `393.wav` | synthetic | train |  | 0.903 |  |  |  |
| `394.wav` | real | train |  | 0.001 |  |  |  |
| `397.wav` | synthetic | train |  | 0.938 |  |  |  |
| `398.wav` | synthetic | train |  | 0.957 |  |  |  |
| `399.wav` | synthetic | train |  | 0.989 |  |  |  |
| `402.wav` | synthetic | train |  | 0.881 |  |  |  |
| `403.wav` | synthetic | train |  | 0.943 |  |  |  |
| `406.wav` | real | train |  | 0.056 |  |  |  |
| `407.wav` | real | train |  | 0.640 |  |  |  |
| `408.wav` | synthetic | train |  | 0.557 |  |  |  |
| `413.wav` | real | train |  | 0.689 |  |  |  |
| `414.wav` | real | train |  | 0.511 |  |  |  |
| `415.wav` | real | train |  | 0.194 |  |  |  |
| `416.wav` | synthetic | train |  | 0.097 |  |  |  |
| `417.wav` | real | train |  | 0.021 |  |  |  |
| `418.wav` | real | train |  | 0.078 |  |  |  |
| `419.wav` | real | train |  | 0.059 |  |  |  |
| `420.wav` | real | train |  | 0.424 |  |  |  |
| `422.wav` | synthetic | train |  | 0.824 |  |  |  |
| `423.wav` | real | train |  | 0.829 |  |  |  |
| `424.wav` | real | train |  | 0.000 |  |  |  |
| `425.wav` | real | train |  | 0.296 |  |  |  |
| `426.wav` | real | train |  | 0.522 |  |  |  |
| `427.wav` | synthetic | train |  | 0.991 |  |  |  |
| `428.wav` | real | train |  | 0.192 |  |  |  |
| `430.wav` | real | train |  | 0.029 |  |  |  |
| `431.wav` | real | train |  | 0.007 |  |  |  |
| `432.wav` | synthetic | train |  | 0.805 |  |  |  |
| `433.wav` | synthetic | train |  | 0.969 |  |  |  |
| `434.wav` | real | train |  | 0.369 |  |  |  |
| `435.wav` | real | train |  | 0.138 |  |  |  |
| `436.wav` | real | train |  | 0.227 |  |  |  |
| `437.wav` | synthetic | train |  | 0.809 |  |  |  |
| `438.wav` | real | train |  | 0.254 |  |  |  |
| `439.wav` | real | train |  | 0.046 |  |  |  |
| `440.wav` | synthetic | train |  | 0.547 |  |  |  |
| `442.wav` | real | train |  | 0.789 |  |  |  |
| `443.wav` | synthetic | train |  | 0.232 |  |  |  |
| `444.wav` | synthetic | train |  | 0.061 |  |  |  |
| `445.wav` | synthetic | train |  | 0.947 |  |  |  |
| `446.wav` | real | train |  | 0.012 |  |  |  |
| `447.wav` | real | train |  | 0.137 |  |  |  |
| `448.wav` | real | train |  | 0.101 |  |  |  |
| `450.wav` | real | train |  | 0.272 |  |  |  |
| `452.wav` | real | train |  | 0.093 |  |  |  |
| `453.wav` | real | train |  | 0.081 |  |  |  |
| `454.wav` | synthetic | train |  | 0.887 |  |  |  |
| `455.wav` | real | train |  | 0.036 |  |  |  |
| `456.wav` | real | train |  | 0.007 |  |  |  |
| `457.wav` | synthetic | train |  | 0.846 |  |  |  |
| `458.wav` | real | train |  | 0.005 |  |  |  |
| `459.wav` | real | train |  | 0.000 |  |  |  |
| `460.wav` | synthetic | train |  | 0.941 |  |  |  |
| `462.wav` | real | train |  | 0.046 |  |  |  |
| `463.wav` | synthetic | train |  | 0.647 |  |  |  |
| `464.wav` | synthetic | train |  | 0.987 |  |  |  |
| `465.wav` | real | train |  | 0.126 |  |  |  |
| `468.wav` | real | train |  | 0.135 |  |  |  |
| `470.wav` | real | train |  | 0.261 |  |  |  |
| `471.wav` | real | train |  | 0.434 |  |  |  |
| `472.wav` | real | train |  | 0.066 |  |  |  |
| `475.wav` | real | train |  | 0.140 |  |  |  |
| `476.wav` | real | train |  | 0.037 |  |  |  |
| `477.wav` | real | train |  | 0.049 |  |  |  |
| `478.wav` | real | train |  | 0.031 |  |  |  |
| `479.wav` | real | train |  | 0.199 |  |  |  |
| `480.wav` | real | train |  | 0.175 |  |  |  |
| `484.wav` | synthetic | train |  | 0.783 |  |  |  |
| `485.wav` | real | train |  | 0.400 |  |  |  |
| `487.wav` | real | train |  | 0.256 |  |  |  |
| `488.wav` | synthetic | train |  | 0.918 |  |  |  |
| `490.wav` | synthetic | train |  | 0.847 |  |  |  |
| `491.wav` | synthetic | train |  | 0.786 |  |  |  |
| `492.wav` | synthetic | train |  | 0.865 |  |  |  |
| `493.wav` | synthetic | train |  | 0.685 |  |  |  |
| `494.wav` | synthetic | train |  | 0.493 |  |  |  |
| `496.wav` | real | train |  | 0.002 |  |  |  |
| `498.wav` | real | train |  | 0.137 |  |  |  |
| `499.wav` | real | train |  | 0.005 |  |  |  |
| `500.wav` | real | train |  | 0.207 |  |  |  |
| `501.wav` | real | train |  | 0.048 |  |  |  |
| `503.wav` | real | train |  | 0.021 |  |  |  |
| `504.wav` | real | train |  | 0.067 |  |  |  |
| `506.wav` | real | train |  | 0.000 |  |  |  |
| `509.wav` | synthetic | train |  | 0.709 |  |  |  |
| `510.wav` | real | train |  | 0.024 |  |  |  |
| `511.wav` | real | train |  | 0.512 |  |  |  |
| `512.wav` | synthetic | train |  | 0.795 |  |  |  |
| `513.wav` | real | train |  | 0.518 |  |  |  |
| `514.wav` | synthetic | train |  | 0.132 |  |  |  |
| `515.wav` | synthetic | train |  | 0.768 |  |  |  |
| `516.wav` | real | train |  | 0.037 |  |  |  |
| `517.wav` | real | train |  | 0.018 |  |  |  |
| `518.wav` | real | train |  | 0.006 |  |  |  |
| `521.wav` | real | train |  | 0.012 |  |  |  |
| `523.wav` | real | train |  | 0.697 |  |  |  |
| `524.wav` | real | train |  | 0.147 |  |  |  |
| `525.wav` | synthetic | train |  | 0.952 |  |  |  |
| `526.wav` | synthetic | train |  | 0.143 |  |  |  |
| `528.wav` | synthetic | train |  | 0.994 |  |  |  |
| `529.wav` | real | train |  | 0.081 |  |  |  |
| `530.wav` | real | train |  | 0.493 |  |  |  |
| `531.wav` | synthetic | train |  | 0.873 |  |  |  |
| `532.wav` | real | train |  | 0.172 |  |  |  |
| `533.wav` | real | train |  | 0.022 |  |  |  |
| `534.wav` | real | train |  | 0.534 |  |  |  |
| `535.wav` | synthetic | train |  | 0.874 |  |  |  |
| `536.wav` | synthetic | train |  | 0.992 |  |  |  |
| `537.wav` | synthetic | train |  | 0.939 |  |  |  |
| `539.wav` | real | train |  | 0.001 |  |  |  |
| `540.wav` | synthetic | train |  | 0.912 |  |  |  |
| `541.wav` | synthetic | train |  | 0.953 |  |  |  |
| `542.wav` | synthetic | train |  | 0.899 |  |  |  |
| `543.wav` | real | train |  | 0.002 |  |  |  |
| `544.wav` | synthetic | train |  | 0.883 |  |  |  |
| `545.wav` | synthetic | train |  | 0.988 |  |  |  |
| `547.wav` | synthetic | train |  | 0.992 |  |  |  |
| `548.wav` | real | train |  | 0.281 |  |  |  |
| `550.wav` | synthetic | train |  | 0.897 |  |  |  |
| `551.wav` | real | train |  | 0.251 |  |  |  |
| `552.wav` | real | train |  | 0.734 |  |  |  |
| `554.wav` | real | train |  | 0.035 |  |  |  |
| `555.wav` | real | train |  | 0.315 |  |  |  |
| `556.wav` | real | train |  | 0.017 |  |  |  |
| `557.wav` | synthetic | train |  | 0.953 |  |  |  |
| `558.wav` | real | train |  | 0.251 |  |  |  |
| `559.wav` | real | train |  | 0.055 |  |  |  |
| `561.wav` | synthetic | train |  | 0.858 |  |  |  |
| `562.wav` | synthetic | train |  | 0.974 |  |  |  |
| `563.wav` | real | train |  | 0.207 |  |  |  |
| `564.wav` | real | train |  | 0.144 |  |  |  |
| `565.wav` | synthetic | train |  | 0.936 |  |  |  |
| `566.wav` | synthetic | train |  | 0.778 |  |  |  |
| `567.wav` | synthetic | train |  | 0.902 |  |  |  |
| `568.wav` | real | train |  | 0.346 |  |  |  |
| `569.wav` | synthetic | train |  | 0.926 |  |  |  |
| `570.wav` | synthetic | train |  | 0.343 |  |  |  |
| `571.wav` | synthetic | train |  | 0.920 |  |  |  |
| `572.wav` | synthetic | train |  | 0.672 |  |  |  |
| `574.wav` | synthetic | train |  | 0.847 |  |  |  |
| `578.wav` | synthetic | train |  | 0.602 |  |  |  |
| `580.wav` | real | train |  | 0.022 |  |  |  |
| `581.wav` | real | train |  | 0.041 |  |  |  |
| `582.wav` | real | train |  | 0.187 |  |  |  |
| `583.wav` | real | train |  | 0.053 |  |  |  |
| `584.wav` | real | train |  | 0.048 |  |  |  |
| `585.wav` | real | train |  | 0.197 |  |  |  |
| `586.wav` | synthetic | train |  | 0.004 |  |  |  |
| `587.wav` | real | train |  | 0.276 |  |  |  |
| `588.wav` | real | train |  | 0.016 |  |  |  |
| `590.wav` | real | train |  | 0.217 |  |  |  |
| `591.wav` | real | train |  | 0.225 |  |  |  |
| `592.wav` | synthetic | train |  | 0.941 |  |  |  |
| `594.wav` | synthetic | train |  | 0.952 |  |  |  |
| `596.wav` | real | train |  | 0.012 |  |  |  |
| `597.wav` | synthetic | train |  | 0.911 |  |  |  |
| `599.wav` | real | train |  | 0.160 |  |  |  |
| `600.wav` | real | train |  | 0.012 |  |  |  |
| `601.wav` | real | train |  | 0.048 |  |  |  |
| `602.wav` | real | train |  | 0.013 |  |  |  |
| `603.wav` | real | train |  | 0.368 |  |  |  |
| `604.wav` | real | train |  | 0.305 |  |  |  |
| `606.wav` | real | train |  | 0.593 |  |  |  |
| `607.wav` | real | train |  | 0.099 |  |  |  |
| `608.wav` | real | train |  | 0.237 |  |  |  |
| `610.wav` | real | train |  | 0.020 |  |  |  |
| `611.wav` | real | train |  | 0.001 |  |  |  |
| `612.wav` | real | train |  | 0.031 |  |  |  |
| `614.wav` | synthetic | train |  | 0.879 |  |  |  |
| `617.wav` | real | train |  | 0.669 |  |  |  |
| `618.wav` | real | train |  | 0.006 |  |  |  |
| `619.wav` | real | train |  | 0.024 |  |  |  |
| `620.wav` | real | train |  | 0.147 |  |  |  |
| `623.wav` | real | train |  | 0.106 |  |  |  |
| `625.wav` | synthetic | train |  | 0.648 |  |  |  |
| `627.wav` | synthetic | train |  | 0.952 |  |  |  |
| `628.wav` | real | train |  | 0.179 |  |  |  |
| `629.wav` | real | train |  | 0.004 |  |  |  |
| `630.wav` | real | train |  | 0.004 |  |  |  |
| `631.wav` | synthetic | train |  | 0.927 |  |  |  |
| `634.wav` | synthetic | train |  | 0.936 |  |  |  |
| `635.wav` | synthetic | train |  | 0.973 |  |  |  |
| `637.wav` | synthetic | train |  | 0.939 |  |  |  |
| `638.wav` | real | train |  | 0.386 |  |  |  |
| `639.wav` | synthetic | train |  | 0.988 |  |  |  |
| `640.wav` | real | train |  | 0.000 |  |  |  |
| `641.wav` | synthetic | train |  | 0.947 |  |  |  |
| `642.wav` | real | train |  | 0.767 |  |  |  |
| `643.wav` | real | train |  | 0.006 |  |  |  |
| `644.wav` | real | train |  | 0.000 |  |  |  |
| `646.wav` | real | train |  | 0.030 |  |  |  |
| `648.wav` | real | train |  | 0.276 |  |  |  |
| `649.wav` | real | train |  | 0.101 |  |  |  |
| `652.wav` | real | train |  | 0.316 |  |  |  |
| `653.wav` | real | train |  | 0.364 |  |  |  |
| `659.wav` | synthetic | train |  | 0.978 |  |  |  |
| `660.wav` | synthetic | train |  | 0.929 |  |  |  |
| `661.wav` | synthetic | train |  | 0.843 |  |  |  |
| `662.wav` | real | train |  | 0.347 |  |  |  |
| `663.wav` | synthetic | train |  | 0.976 |  |  |  |
| `664.wav` | synthetic | train |  | 0.635 |  |  |  |
| `665.wav` | real | train |  | 0.236 |  |  |  |
| `666.wav` | real | train |  | 0.062 |  |  |  |
| `667.wav` | real | train |  | 0.378 |  |  |  |
| `669.wav` | real | train |  | 0.083 |  |  |  |
| `670.wav` | real | train |  | 0.001 |  |  |  |
| `671.wav` | real | train |  | 0.090 |  |  |  |
| `672.wav` | real | train |  | 0.012 |  |  |  |
| `673.wav` | real | train |  | 0.116 |  |  |  |
| `674.wav` | real | train |  | 0.444 |  |  |  |
| `675.wav` | real | train |  | 0.025 |  |  |  |
| `676.wav` | real | train |  | 0.382 |  |  |  |
| `679.wav` | real | train |  | 0.554 |  |  |  |
| `680.wav` | synthetic | train |  | 0.977 |  |  |  |
| `681.wav` | synthetic | train |  | 0.628 |  |  |  |
| `682.wav` | synthetic | train |  | 0.217 |  |  |  |
| `684.wav` | real | train |  | 0.547 |  |  |  |
| `685.wav` | synthetic | train |  | 0.862 |  |  |  |
| `687.wav` | synthetic | train |  | 0.621 |  |  |  |
| `688.wav` | synthetic | train |  | 0.651 |  |  |  |
| `689.wav` | real | train |  | 0.198 |  |  |  |
| `690.wav` | synthetic | train |  | 0.969 |  |  |  |
| `692.wav` | synthetic | train |  | 0.622 |  |  |  |
| `693.wav` | real | train |  | 0.118 |  |  |  |
| `694.wav` | real | train |  | 0.655 |  |  |  |
| `695.wav` | synthetic | train |  | 0.942 |  |  |  |
| `696.wav` | synthetic | train |  | 0.701 |  |  |  |
| `697.wav` | real | train |  | 0.023 |  |  |  |
| `698.wav` | real | train |  | 0.091 |  |  |  |
| `699.wav` | real | train |  | 0.143 |  |  |  |
| `700.wav` | real | train |  | 0.313 |  |  |  |
| `701.wav` | real | train |  | 0.323 |  |  |  |
| `702.wav` | real | train |  | 0.004 |  |  |  |
| `704.wav` | synthetic | train |  | 0.866 |  |  |  |
| `705.wav` | real | train |  | 0.744 |  |  |  |
| `706.wav` | real | train |  | 0.004 |  |  |  |
| `707.wav` | real | train |  | 0.393 |  |  |  |
| `708.wav` | synthetic | train |  | 0.935 |  |  |  |
| `709.wav` | synthetic | train |  | 0.946 |  |  |  |
| `710.wav` | real | train |  | 0.289 |  |  |  |
| `711.wav` | real | train |  | 0.280 |  |  |  |
| `712.wav` | synthetic | train |  | 0.913 |  |  |  |
| `714.wav` | real | train |  | 0.656 |  |  |  |
| `718.wav` | synthetic | train |  | 0.342 |  |  |  |
| `719.wav` | real | train |  | 0.753 |  |  |  |
| `720.wav` | synthetic | train |  | 0.943 |  |  |  |
| `721.wav` | real | train |  | 0.193 |  |  |  |
| `722.wav` | synthetic | train |  | 0.671 |  |  |  |
| `723.wav` | real | train |  | 0.003 |  |  |  |
| `725.wav` | real | train |  | 0.034 |  |  |  |
| `726.wav` | real | train |  | 0.122 |  |  |  |
| `727.wav` | real | train |  | 0.176 |  |  |  |
| `728.wav` | synthetic | train |  | 0.023 |  |  |  |
| `730.wav` | real | train |  | 0.477 |  |  |  |
| `732.wav` | synthetic | train |  | 0.822 |  |  |  |
| `734.wav` | real | train |  | 0.006 |  |  |  |
| `736.wav` | real | train |  | 0.451 |  |  |  |
| `737.wav` | synthetic | train |  | 0.778 |  |  |  |
| `738.wav` | real | train |  | 0.535 |  |  |  |
| `740.wav` | real | train |  | 0.600 |  |  |  |
| `741.wav` | real | train |  | 0.026 |  |  |  |
| `742.wav` | real | train |  | 0.010 |  |  |  |
| `743.wav` | synthetic | train |  | 0.888 |  |  |  |
| `746.wav` | real | train |  | 0.020 |  |  |  |
| `747.wav` | real | train |  | 0.359 |  |  |  |
| `748.wav` | real | train |  | 0.223 |  |  |  |
| `749.wav` | synthetic | train |  | 0.901 |  |  |  |
| `750.wav` | real | train |  | 0.129 |  |  |  |
| `751.wav` | real | train |  | 0.101 |  |  |  |
| `752.wav` | real | train |  | 0.029 |  |  |  |
| `753.wav` | synthetic | train |  | 0.947 |  |  |  |
| `754.wav` | real | train |  | 0.102 |  |  |  |
| `755.wav` | real | train |  | 0.044 |  |  |  |
| `756.wav` | synthetic | train |  | 0.901 |  |  |  |
| `758.wav` | synthetic | train |  | 0.681 |  |  |  |
| `760.wav` | real | train |  | 0.393 |  |  |  |
| `761.wav` | real | train |  | 0.032 |  |  |  |
| `762.wav` | real | train |  | 0.001 |  |  |  |
| `763.wav` | synthetic | train |  | 0.730 |  |  |  |
| `765.wav` | real | train |  | 0.420 |  |  |  |
| `766.wav` | real | train |  | 0.766 |  |  |  |
| `767.wav` | real | train |  | 0.714 |  |  |  |
| `768.wav` | synthetic | train |  | 0.814 |  |  |  |
| `771.wav` | real | train |  | 0.103 |  |  |  |
| `773.wav` | synthetic | train |  | 0.960 |  |  |  |
| `774.wav` | real | train |  | 0.013 |  |  |  |
| `775.wav` | real | train |  | 0.161 |  |  |  |
| `776.wav` | real | train |  | 0.061 |  |  |  |
| `777.wav` | real | train |  | 0.960 |  |  |  |
| `778.wav` | real | train |  | 0.001 |  |  |  |
| `779.wav` | real | train |  | 0.093 |  |  |  |
| `781.wav` | synthetic | train |  | 0.955 |  |  |  |
| `782.wav` | synthetic | train |  | 0.974 |  |  |  |
| `783.wav` | synthetic | train |  | 0.890 |  |  |  |
| `784.wav` | synthetic | train |  | 0.974 |  |  |  |
| `787.wav` | synthetic | train |  | 0.973 |  |  |  |
| `795.wav` | synthetic | train |  | 0.940 |  |  |  |
| `797.wav` | synthetic | train |  | 0.877 |  |  |  |
| `798.wav` | synthetic | train |  | 0.826 |  |  |  |
| `800.wav` | synthetic | train |  | 0.825 |  |  |  |
| `802.wav` | synthetic | train |  | 0.751 |  |  |  |
| `803.wav` | synthetic | train |  | 0.988 |  |  |  |
| `807.wav` | synthetic | train |  | 0.970 |  |  |  |
| `809.wav` | synthetic | train |  | 0.106 |  |  |  |
| `811.wav` | synthetic | train |  | 0.182 |  |  |  |
| `816.wav` | synthetic | train |  | 0.542 |  |  |  |
| `820.wav` | synthetic | train |  | 0.291 |  |  |  |
| `827.wav` | synthetic | train |  | 0.005 |  |  |  |
| `834.wav` | synthetic | train |  | 0.913 |  |  |  |
| `836.wav` | synthetic | train |  | 0.913 |  |  |  |
| `838.wav` | synthetic | train |  | 0.596 |  |  |  |
| `839.wav` | synthetic | train |  | 0.961 |  |  |  |
| `841.wav` | synthetic | train |  | 0.780 |  |  |  |
| `842.wav` | synthetic | train |  | 0.898 |  |  |  |
| `843.wav` | synthetic | train |  | 0.982 |  |  |  |
| `848.wav` | synthetic | train |  | 0.969 |  |  |  |
| `856.wav` | synthetic | train |  | 0.729 |  |  |  |
| `857.wav` | synthetic | train |  | 0.598 |  |  |  |
| `865.wav` | synthetic | train |  | 0.929 |  |  |  |
| `868.wav` | synthetic | train |  | 0.984 |  |  |  |
| `870.wav` | synthetic | train |  | 0.984 |  |  |  |
| `873.wav` | synthetic | train |  | 0.775 |  |  |  |
| `877.wav` | synthetic | train |  | 0.970 |  |  |  |
| `879.wav` | synthetic | train |  | 0.737 |  |  |  |
| `882.wav` | synthetic | train |  | 0.936 |  |  |  |
| `883.wav` | synthetic | train |  | 0.892 |  |  |  |
| `884.wav` | synthetic | train |  | 0.471 |  |  |  |
| `888.wav` | synthetic | train |  | 0.987 |  |  |  |
| `889.wav` | synthetic | train |  | 0.439 |  |  |  |
| `891.wav` | synthetic | train |  | 0.927 |  |  |  |
| `898.wav` | synthetic | train |  | 0.886 |  |  |  |
| `901.wav` | synthetic | train |  | 0.973 |  |  |  |
| `904.wav` | synthetic | train |  | 0.992 |  |  |  |
| `905.wav` | synthetic | train |  | 0.874 |  |  |  |
| `908.wav` | synthetic | train |  | 0.917 |  |  |  |
| `909.wav` | synthetic | train |  | 0.969 |  |  |  |
| `916.wav` | synthetic | train |  | 0.998 |  |  |  |
| `924.wav` | synthetic | train |  | 0.874 |  |  |  |
| `933.wav` | synthetic | train |  | 0.859 |  |  |  |
| `938.wav` | synthetic | train |  | 0.981 |  |  |  |
| `940.wav` | synthetic | train |  | 0.710 |  |  |  |
| `942.wav` | synthetic | train |  | 0.986 |  |  |  |
| `944.wav` | synthetic | train |  | 0.919 |  |  |  |
| `950.wav` | synthetic | train |  | 0.886 |  |  |  |
| `955.wav` | synthetic | train |  | 0.983 |  |  |  |
| `957.wav` | synthetic | train |  | 0.943 |  |  |  |
| `963.wav` | synthetic | train |  | 0.080 |  |  |  |
| `967.wav` | synthetic | train |  | 0.983 |  |  |  |
| `968.wav` | synthetic | train |  | 0.180 |  |  |  |
| `978.wav` | synthetic | train |  | 0.892 |  |  |  |
| `979.wav` | synthetic | train |  | 0.959 |  |  |  |
| `980.wav` | synthetic | train |  | 0.947 |  |  |  |
| `982.wav` | synthetic | train |  | 0.650 |  |  |  |
| `983.wav` | synthetic | train |  | 0.486 |  |  |  |
| `990.wav` | synthetic | train |  | 0.992 |  |  |  |
| `991.wav` | synthetic | train |  | 0.948 |  |  |  |
| `1003.wav` | synthetic | train |  | 0.966 |  |  |  |
| `1006.wav` | synthetic | train |  | 0.933 |  |  |  |
| `1013.wav` | synthetic | train |  | 0.971 |  |  |  |
| `1015.wav` | synthetic | train |  | 0.841 |  |  |  |
| `1017.wav` | synthetic | train |  | 0.768 |  |  |  |
| `1021.wav` | synthetic | train |  | 0.860 |  |  |  |
| `1028.wav` | synthetic | train |  | 0.779 |  |  |  |
| `1033.wav` | synthetic | train |  | 0.953 |  |  |  |
| `1034.wav` | synthetic | train |  | 0.756 |  |  |  |
| `1059.wav` | synthetic | train |  | 0.124 |  |  |  |
| `1061.wav` | synthetic | train |  | 0.868 |  |  |  |
| `1064.wav` | synthetic | train |  | 0.938 |  |  |  |
| `1068.wav` | synthetic | train |  | 0.799 |  |  |  |
| `1069.wav` | synthetic | train |  | 0.967 |  |  |  |
| `1071.wav` | synthetic | train |  | 0.953 |  |  |  |
| `1073.wav` | synthetic | train |  | 0.984 |  |  |  |
| `1078.wav` | synthetic | train |  | 0.849 |  |  |  |
| `1087.wav` | synthetic | train |  | 0.815 |  |  |  |
| `1090.wav` | synthetic | train |  | 0.842 |  |  |  |
| `1092.wav` | synthetic | train |  | 0.907 |  |  |  |
| `1095.wav` | synthetic | train |  | 0.843 |  |  |  |
| `1097.wav` | synthetic | train |  | 0.971 |  |  |  |
| `1100.wav` | synthetic | train |  | 0.874 |  |  |  |
| `1105.wav` | synthetic | train |  | 0.896 |  |  |  |
| `1116.wav` | synthetic | train |  | 0.354 |  |  |  |
| `1117.wav` | synthetic | train |  | 0.546 |  |  |  |
| `1121.wav` | synthetic | train |  | 0.923 |  |  |  |
| `1123.wav` | synthetic | train |  | 0.693 |  |  |  |
| `1126.wav` | synthetic | train |  | 0.506 |  |  |  |
| `1127.wav` | synthetic | train |  | 0.760 |  |  |  |
| `1128.wav` | synthetic | train |  | 0.996 |  |  |  |
| `1132.wav` | synthetic | train |  | 0.953 |  |  |  |
| `1139.wav` | synthetic | train |  | 0.926 |  |  |  |
| `1144.wav` | synthetic | train |  | 0.895 |  |  |  |
| `1145.wav` | synthetic | train |  | 0.871 |  |  |  |
| `1148.wav` | synthetic | train |  | 0.918 |  |  |  |
| `1160.wav` | synthetic | train |  | 0.898 |  |  |  |
| `1161.wav` | synthetic | train |  | 0.956 |  |  |  |
| `1165.wav` | synthetic | train |  | 0.644 |  |  |  |
| `1169.wav` | synthetic | train |  | 0.960 |  |  |  |
| `1170.wav` | synthetic | train |  | 0.969 |  |  |  |
| `1171.wav` | synthetic | train |  | 0.789 |  |  |  |
| `1177.wav` | synthetic | train |  | 0.831 |  |  |  |
| `1181.wav` | synthetic | train |  | 0.990 |  |  |  |
| `1182.wav` | synthetic | train |  | 0.071 |  |  |  |
| `1183.wav` | synthetic | train |  | 0.898 |  |  |  |
| `1186.wav` | synthetic | train |  | 0.328 |  |  |  |
| `1195.wav` | synthetic | train |  | 0.954 |  |  |  |
| `1197.wav` | synthetic | train |  | 0.055 |  |  |  |
| `1198.wav` | synthetic | train |  | 0.975 |  |  |  |
| `1202.wav` | synthetic | train |  | 0.983 |  |  |  |
| `1203.wav` | synthetic | train |  | 0.965 |  |  |  |
| `1205.wav` | synthetic | train |  | 0.883 |  |  |  |
| `1207.wav` | synthetic | train |  | 0.845 |  |  |  |
| `1208.wav` | synthetic | train |  | 0.365 |  |  |  |
| `1212.wav` | synthetic | train |  | 0.966 |  |  |  |
| `1217.wav` | synthetic | train |  | 0.909 |  |  |  |
| `1222.wav` | synthetic | train |  | 0.626 |  |  |  |
| `1223.wav` | synthetic | train |  | 0.847 |  |  |  |
| `1224.wav` | synthetic | train |  | 0.859 |  |  |  |
| `1226.wav` | synthetic | train |  | 0.820 |  |  |  |
| `1228.wav` | synthetic | train |  | 0.961 |  |  |  |
| `1229.wav` | synthetic | train |  | 0.953 |  |  |  |
| `1231.wav` | synthetic | train |  | 0.977 |  |  |  |
| `1234.wav` | synthetic | train |  | 0.358 |  |  |  |
| `1239.wav` | synthetic | train |  | 0.750 |  |  |  |
| `1240.wav` | synthetic | train |  | 0.969 |  |  |  |
| `1242.wav` | synthetic | train |  | 0.867 |  |  |  |
| `1244.wav` | synthetic | train |  | 0.972 |  |  |  |
| `1246.wav` | synthetic | train |  | 0.947 |  |  |  |
| `1249.wav` | synthetic | train |  | 0.929 |  |  |  |
| `1251.wav` | synthetic | train |  | 0.948 |  |  |  |
| `1252.wav` | synthetic | train |  | 0.194 |  |  |  |
| `1253.wav` | synthetic | train |  | 0.779 |  |  |  |
| `1254.wav` | synthetic | train |  | 0.950 |  |  |  |
| `1264.wav` | synthetic | train |  | 0.006 |  |  |  |
| `1266.wav` | synthetic | train |  | 0.840 |  |  |  |
| `1270.wav` | synthetic | train |  | 0.828 |  |  |  |
| `1273.wav` | synthetic | train |  | 0.964 |  |  |  |
| `1274.wav` | synthetic | train |  | 0.852 |  |  |  |
| `1279.wav` | synthetic | train |  | 0.994 |  |  |  |
| `1281.wav` | synthetic | train |  | 0.046 |  |  |  |
| `1283.wav` | synthetic | train |  | 0.723 |  |  |  |
| `1293.wav` | synthetic | train |  | 0.942 |  |  |  |
| `1302.wav` | synthetic | train |  | 0.016 |  |  |  |
| `1307.wav` | synthetic | train |  | 0.994 |  |  |  |
| `1308.wav` | synthetic | train |  | 0.855 |  |  |  |
| `1309.wav` | synthetic | train |  | 0.964 |  |  |  |
| `1316.wav` | synthetic | train |  | 0.730 |  |  |  |
| `1335.wav` | synthetic | train |  | 0.960 |  |  |  |
| `1336.wav` | synthetic | train |  | 0.968 |  |  |  |
| `1338.wav` | synthetic | train |  | 0.870 |  |  |  |
| `1339.wav` | synthetic | train |  | 0.980 |  |  |  |
| `1343.wav` | synthetic | train |  | 0.898 |  |  |  |
| `1344.wav` | synthetic | train |  | 0.857 |  |  |  |
| `1348.wav` | synthetic | train |  | 0.965 |  |  |  |
| `1351.wav` | synthetic | train |  | 0.952 |  |  |  |
| `1353.wav` | synthetic | train |  | 0.877 |  |  |  |
| `1355.wav` | synthetic | train |  | 0.524 |  |  |  |
| `1359.wav` | synthetic | train |  | 0.928 |  |  |  |

## Operating Bands

- Scores are calibrated on held-out predictions, not in-sample predictions.
- Max real score: 0.907; min synthetic score: 0.000; nearest detected synthetic above the real range: 0.908.
- Suggested legacy WARN threshold: 0.000. Suggested legacy BLOCK threshold: 0.908.
- At BLOCK threshold: thr=0.908, TPR=0.54, FPR=0.00, TP=67, FP=0, TN=125, FN=58.
- At catch-all synthetic threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0.

## Feature-Set Ablations

| feature set | zero-FP result | max real | min synthetic | suggested block | catch-all result |
|---|---|---:|---:|---:|---|
| `all` | thr=0.908, TPR=0.54, FPR=0.00, TP=67, FP=0, TN=125, FN=58 | 0.907 | 0.000 | 0.908 | thr=0.000, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0 |
| `local+speaker_fast` | thr=0.908, TPR=0.54, FPR=0.00, TP=67, FP=0, TN=125, FN=58 | 0.907 | 0.000 | 0.908 | thr=0.000, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0 |
| `local_only_fast` | thr=0.908, TPR=0.54, FPR=0.00, TP=67, FP=0, TN=125, FN=58 | 0.907 | 0.000 | 0.908 | thr=0.000, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0 |
| `citadel_only` | none | 0.500 | 0.500 | 0.500 | thr=0.500, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0 |
| `speaker_only` | none | 0.500 | 0.500 | 0.500 | thr=0.500, TPR=1.00, FPR=1.00, TP=125, FP=125, TN=0, FN=0 |

## Group Holdouts

| held-out column | mode | groups | covered samples | skipped groups | zero-FP result |
|---|---|---:|---:|---:|---|
| `dataset` | split | 0 | 0 | 0 | need at least two groups |
| `generator_id` | split | 0 | 0 | 0 | could not find a group split with both classes in train and test |
| `speaker_id` | split | 53 | 208 | 0 | thr=0.963, TPR=0.22, FPR=0.00, TP=26, FP=0, TN=88, FN=94 |

## Current-Signal Baselines

- `sig_deepfake_detection` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=500, FP=500, TN=0, FN=0
- `citadel_risk` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=500, FP=500, TN=0, FN=0
- `sig_wavlm_paralinguistic` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=500, FP=500, TN=0, FN=0
- `sig_rmt_divergence` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=500, FP=500, TN=0, FN=0

## Largest Coefficients

| feature | weight |
|---|---:|
| `flatness_mean` | -3.373 |
| `rms_std` | -1.531 |
| `zcr_mean` | 0.776 |
| `hf_ratio_mean` | 0.690 |
| `silence_ratio` | 0.591 |
| `centroid_mean` | -0.496 |
| `pitch_std` | -0.477 |
| `pitch_mean` | 0.455 |
| `centroid_std` | 0.357 |
| `bias` | -0.319 |
| `rms_mean` | 0.253 |
| `flatness_std` | 0.191 |

## Interpretation

This experiment can show whether the available features contain signal, but it does not prove generalization. A deployable detector needs more real speakers, more clone providers, and provider-held-out evaluation.

## Model Artifact

```json
{
  "features": [
    "speaker_cosine_vs_real_01",
    "citadel_risk",
    "sig_spectral_anomaly",
    "sig_steganalysis",
    "sig_wavlm_paralinguistic",
    "sig_deepfake_detection",
    "sig_waveform_integrity",
    "sig_rmt_divergence",
    "sig_asr_prefix_attack",
    "rms_mean",
    "rms_std",
    "silence_ratio",
    "centroid_mean",
    "centroid_std",
    "flatness_mean",
    "flatness_std",
    "hf_ratio_mean",
    "zcr_mean",
    "pitch_mean",
    "pitch_std",
    "pitch_cv"
  ],
  "weights": [
    -0.3193148755382613,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.25279462828879606,
    -1.5306845275159717,
    0.5908425750303878,
    -0.4958205524179659,
    0.35698480879622985,
    -3.372936020833733,
    0.19070582882554193,
    0.6903981913520812,
    0.7755395934435888,
    0.4551178208731569,
    -0.4768114438460158,
    -0.06928676022008101
  ],
  "mean": [
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.1006930270000001,
    0.09297984400000003,
    0.20770214699999984,
    757.9759520000001,
    812.5499630000006,
    0.153139172,
    0.10537500200000001,
    0.04345924699999998,
    0.10427056400000011,
    173.94100200000005,
    61.742413,
    0.36174375599999975
  ],
  "std": [
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    0.03623231821921239,
    0.025762344941671424,
    0.1288602366505951,
    323.0538211903797,
    501.40480822332165,
    0.08420856812354917,
    0.044903241445200776,
    0.045898582689294344,
    0.04187345866330969,
    39.9475208470813,
    29.748253826744698,
    0.161704605022295
  ],
  "evaluation": "stratified holdout split, test_size=0.25, seed=7."
}
```
