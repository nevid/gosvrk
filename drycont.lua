-- ������� �������� �������� ���. ��������� 
-- ������������ ����� �����. ������ � ������������
--  1=fail 0=OK
local ipa_ionm={}
--digdata_ark["k2mp1"]=0;
--digdata_ark["pered"]=0;
--digdata_ark["unn1powerok"]=0;
--digdata_ark["unn1batteryok"]=0;
--digdata_ark["k2mp2"]=0;
--digdata_ark["zad"]=0;
--digdata_ark["unn2powerok"]=0;
--digdata_ark["unn2batteryok"]=0;

--������� ������������ ������� ���. �������� (����) � ��������� IPA � ��� ����� (������� �� 0 !)
local ipa_io={}
--ipa_io[13]="pered";
ipa_io[8*5+2]="pered";
--ipa_io[14]="zad";
ipa_io[8*5+1]="zad";

ipa_io[28]="clemn1";
ipa_io[29]="clemn2";

ipa_io[43]="temper";

ipa_io[62]="unn1powerok";
ipa_io[63]="unn1batteryok";
ipa_io[64]="k1mp1";
ipa_io[65]="k2mp1";
ipa_io[66]="k3mp1";
ipa_io[67]="k4mp1";
ipa_io[68]="k5mp1";

ipa_io[85]="unn2powerok";
ipa_io[86]="unn2batteryok";
ipa_io[87]="k1mp2";
ipa_io[88]="k2mp2";
ipa_io[89]="k3mp2";
ipa_io[90]="k4mp2";
ipa_io[91]="k5mp2";

--ipa_io[14]="pozhar1";
--ipa_io[15]="pozhar2";

--���. ����. �����. ���������� 
--[���][���(� ��������� � ���������. ���.:1-13)]
ipa_io_mod={
     {0,1,2,3,4,5,6,7,8,9,10,11,12},
	 {15,16,17,18,19,20,21,22,23,24,25,26,27},
     {30,31,32,33,34,35,36,37,38,39,40,41,42},
     {49,50,51,52,53,54,55,56,57,58,59,60,61},
     {72,73,74,75,76,77,78,79,80,81,82,83,84},
};

----------------------



--������������� �������� ���. �������� � �������
--n=������ � ������� ipa_io
function Ipa_SetDigData(n,val)
	if(ipa_io[n]==nil) then return; end
	ipa_ionm[ipa_io[n]]=val;  
end

-- �����. ����. ���. ����. �� ��� �����
-- dig_mame = ��� ���. ����. (� ������� digdata_ark)
-- ret: -1=����. ���. ����. ����������
function Ipa_GetDigData(dig_name)
	if(ipa_ionm[dig_name]==nil) then return -1; end
	return ipa_ionm[dig_name];
end

--�����. ��� ������� dig_name �� ������� ������� n, ����� ����� �� GetDigData_Ipa �������� �������
function Ipa_GetDigDataNm(n)
	if(ipa_io[n]==nil) then return ""; end
	return ipa_io[n];
end

----------------------------------

--local t=GetDigData_Ark("pered");
--print (t);

